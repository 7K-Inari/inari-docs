# 12. Extension OIDC pass-through via token exchange

- Status: Proposed
- Date: 2026-09-26
- Deciders: Inari platform engineering
- Source: [Inari platform plan](../architecture/inari-platform-plan.md) §5.8 (extension host), §5.4 (IAM/OIDC model); extends the `/api/extensions/<name>/*` proxy-extension contract

## Context

Plan §5.8 establishes the extension host: backend plugins run as gRPC sidecars and their HTTP endpoints surface through an authenticated reverse-proxy path (`/api/extensions/<name>/*`) with RBAC enforcement (`extensions, invoke, <name>`). What §5.8 does **not** define is how an extension authenticates to its *downstream* service (ArgoCD, AWS APIs, third-party systems) when acting on behalf of the logged-in user. Without a decision here, each extension would invent its own credential handling — the exact inconsistency the "small kernel, everything else extension" principle is meant to avoid.

Requirements:

- Downstream calls must carry **user-level identity** so audit trails and authorization in the downstream service attribute actions to the real user, not to a shared robot.
- The user's platform access token must **not** be forwarded verbatim to downstream services (audience mismatch, token leakage beyond its intended scope).
- Extension authors need a **uniform, minimal contract**; first-party extensions (`inari-ext-argocd`) dogfood the same SDK (§5.8).
- ArgoCD is the reference first-party extension and has a constraint generic OIDC cannot satisfy directly: **ArgoCD only accepts ArgoCD-issued JWTs** (its own session tokens), not externally-issued OIDC access tokens.

Alternatives considered:

1. **Forward the user's platform token verbatim to downstream services.** Rejected: audience is wrong (token is minted for the Inari API), it leaks a broadly-scoped credential to every downstream hop, and revocation/expiry semantics diverge per service.
2. **Service accounts only, per extension.** Rejected as the *default*: all user actions collapse into one robot identity downstream — no per-user audit, no per-user RBAC in the downstream service. Retained as a pluggable method for services with no user concept.
3. **Full OAuth broker per extension** (each extension runs its own authorization-code flow per downstream service). Rejected: N×M complexity, per-service app registration burden, token stores scattered across extensions; the extension host is the natural choke point to do this once.
4. **Token exchange at the extension host (chosen).** One mechanism, centralized policy, user identity preserved, downstream tokens narrowly audience-scoped.

## Decision

We will make **OIDC user-token pass-through via RFC 8693 token exchange the default extension authentication method**, implemented in the extension host behind a `ConnectionProvider` abstraction.

### Generic OIDC method: RFC 8693 token exchange

- On each proxied extension request, the extension host exchanges the caller's platform access token for an **audience-scoped token** for the extension's downstream service (`requested_token_type=urn:ietf:params:oauth:token-type:access_token`, audience = the downstream service's Keycloak client).
- The flow is **stateless per request by default**: no per-user session is created in the control plane for the generic method.
- Exchanged tokens are **cached in memory with a maximum TTL of 60 seconds** (bounded by the token's own expiry, whichever is shorter) and are **never persisted** to disk or database. Cache keys include user, extension, audience, and tenant; a 401 from the downstream service evicts the entry immediately.

### Header hygiene: identity vs. downstream credentials

- The extensionhost proxy **strips the incoming `Authorization` header** before forwarding and **injects the downstream credential** resolved by the connection provider.
- User identity remains in the request-scoped **`AuthContext`** (claims, tenant, RBAC decision already made at the gateway/extension host). Downstream credentials are a **separate concern** and are never conflated with identity: an extension reads *who* the user is from `AuthContext`, and receives *how* to authenticate downstream from its declared connection method.

### ConnectionProvider abstraction

- Each extension declares a **connection method** in its manifest; the extension host resolves credentials through a pluggable `ConnectionProvider` interface:
  - `oidc` (**default**) — RFC 8693 token exchange as above.
  - `service-account` — a per-extension, per-tenant service account credential delivered via ESO.
  - `api-key` — a static API key referenced by ESO-backed secret (never stored in the database).
  - `shared-secret` — HMAC-style shared-secret signing for legacy systems.
- Extension authors never handle raw user tokens; the SDK exposes the resolved downstream credential and the `AuthContext` separately.

### ArgoCD special case: `oidc-sso-session` method

ArgoCD accepts only **ArgoCD-issued JWTs**, so plain token exchange does not apply. The ArgoCD extension therefore uses a dedicated `oidc-sso-session` connection method:

- **Dex per tenant cluster**, federating to the platform Keycloak as its upstream identity provider. Dex is installed as part of the tenant-zone baseline alongside ArgoCD (§5.3).
- **One `cluster-<id>-dex` OIDC client per cluster** in Keycloak, mirroring the per-cluster agent client model (§5.3) — revocation and rotation stay per-cluster.
- **Zero-prompt SSO bootstrap**: because the user already holds a platform Keycloak session, the console → Dex → Keycloak → ArgoCD round-trip completes without any interactive login prompt, yielding an ArgoCD session JWT scoped to the user.
- ArgoCD session state is stored control-plane-side in an **encrypted `user_extension_sessions`** table (user, tenant, cluster, extension, session handle, expiry) so repeat requests can reuse a live ArgoCD session instead of re-bootstrapping.
- **Agent hop — credential vault + redemption**: ArgoCD ultimately lives in the tenant cluster, reachable only over the agent stream (§5.3). Session credentials are deposited in a control-plane **credential vault**, and the agent receives a **short-lived, single-use redemption token** scoped to one operation; redeeming it retrieves the session credential just-in-time for the tunneled ArgoCD call. No ArgoCD session material is stored on the agent or the cluster.
- **Verification gate for direct external-issuer support**: if/when ArgoCD (or a proxy in front of it) can validate externally-issued tokens directly, that path ships only after a **verification gate** — an explicit end-to-end security review and soak test proving audience, expiry, and RBAC-mapping behavior — and until then `oidc-sso-session` remains the only supported method for ArgoCD.

### End-to-end flows

Generic OIDC extension (token exchange):

```mermaid
sequenceDiagram
    participant U as User (Console/CLI)
    participant GW as API Gateway / BFF
    participant EH as Extension Host
    participant KC as Platform Keycloak
    participant EXT as Extension (sidecar)
    participant DS as Downstream Service

    U->>GW: Request + platform access token
    GW->>EH: Authenticated request (AuthContext: user, tenant, RBAC)
    EH->>KC: RFC 8693 token exchange (subject token, target audience)
    KC-->>EH: Audience-scoped token
    EH->>EH: Cache token (in-memory, TTL ≤ 60s)
    EH->>EXT: Proxied call (Authorization stripped; AuthContext attached)
    EXT->>DS: Downstream call with exchanged token
    DS-->>U: Response (via EXT/EH/GW)
```

ArgoCD extension (`oidc-sso-session`):

```mermaid
sequenceDiagram
    participant U as User (Console)
    participant GW as API Gateway / BFF
    participant EH as Extension Host
    participant DEX as Dex (tenant cluster)
    participant KC as Platform Keycloak
    participant DB as user_extension_sessions (encrypted)
    participant V as Credential Vault
    participant AG as inari-agent
    participant AC as Tenant-local ArgoCD

    U->>GW: ArgoCD action (e.g. sync)
    GW->>EH: Authenticated request (AuthContext)
    EH->>DB: Look up live session (user, tenant, cluster)
    alt no live session
        EH->>DEX: SSO bootstrap via cluster-<id>-dex client
        DEX->>KC: Federated login (existing platform session)
        KC-->>DEX: ID token (zero prompt)
        DEX-->>AC: Callback
        AC-->>EH: ArgoCD-issued session JWT
        EH->>DB: Store encrypted session handle
    end
    EH->>V: Deposit session credential; issue short-lived redemption token
    EH->>AG: Command over agent stream + redemption token
    AG->>V: Redeem (single-use, short TTL)
    V-->>AG: ArgoCD session credential
    AG->>AC: ArgoCD API call as the user
    AC-->>U: Result (via AG/EH/GW)
```

### Security and failure semantics

- **Fail closed**: if token exchange or SSO bootstrap fails, the request fails — there is **no silent fallback to a service-account credential** for user-scoped calls.
- **Audience isolation**: exchanged tokens are accepted only by the audience they were minted for; the extension host validates extension ↔ audience binding from the manifest, preventing an extension from requesting tokens for another extension's service.
- **Cache discipline**: in-memory only, ≤ 60s TTL, immediate eviction on downstream 401; nothing sensitive survives a control-plane restart.
- **Revocation and logout**: control-plane logout or Keycloak session revocation invalidates the exchange path immediately (subject token rejected); `user_extension_sessions` rows are deleted on logout and expired rows are reaped, and the ArgoCD session is terminated best-effort.
- **No tokens in logs or audit payloads**: audit events record actor, extension, method, audience, and downstream status — never token material.
- **Blast radius**: a compromised extension sidecar sees only audience-scoped, ≤ 60s tokens for its own downstream service, never the user's platform token (stripped at the proxy).

## Consequences

- **Easier**: uniform auth contract for all extension authors (first-party included); per-user attribution and RBAC enforcement in downstream services; one audited choke point for credential issuance; ArgoCD SSO works without extra logins while keeping per-cluster isolation via `cluster-<id>-dex` clients.
- **Harder**: Dex becomes a new baseline component per tenant cluster (install, upgrade, version-skew policy like the rest of the bundle); `user_extension_sessions` introduces an encrypted session store to operate and back up; token exchange adds latency, mitigated by the 60s cache; the vault/redemption hop adds moving parts to the agent command path.
- **Follow-up work**: `ConnectionProvider` SDK contract and manifest schema; `user_extension_sessions` encryption design (aligned with [ADR-0013](0013-per-user-git-connections-model-c.md) envelope-encryption approach); Dex bundle packaging in the tenant-zone baseline; verification-gate checklist for direct external-issuer support.
- **We would revisit if**: ArgoCD gains first-class external OIDC issuer validation that passes the verification gate (making Dex optional), or if a SPIFFE/mTLS-based extension identity model supersedes bearer-token exchange across trust zones.
