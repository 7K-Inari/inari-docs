# 10. Hybrid multi-tenant GitHub App model for git credentials

- Status: Accepted
- Date: 2026-09-13
- Deciders: Inari platform engineering
- Source: [Inari platform plan](../architecture/inari-platform-plan.md) §12.1/§12.2, extends [ADR-0004](0004-platform-owned-tenant-state-repos.md)

## Context

ADR-0004 established platform-owned `<tenant>-inari-state` repositories with GitHub App credentials delivered via ESO. The initial implementation pinned the control plane to exactly one `INARI_GITHUB_APP_ID` + one `INARI_GITHUB_APP_INSTALLATION_ID`, which binds every tenant's state repo to a single GitHub organization. Tenants whose state repos live in their own orgs (the common case) could not be served, and the live e2e run confirmed deploys fail without a per-tenant git config + working app credentials.

## Decision

Adopt a **hybrid, two-model credential architecture**, implemented A-then-B:

- **Model A (default): one platform GitHub App, per-tenant installations.** The pinned installation ID is dropped. At deploy time the orchestrator resolves the platform app's installation for the org that owns the tenant's state repo (`GET /app/installations`, matched on `account.login`), cached with a TTL plus invalidate-on-401/404 semantics. Tenants install the platform app into their org **repo-scoped to their `<tenant>-inari-state` repo only**. A missing installation fails the deploy with a 412 carrying the app install link.
- **Model B (override): BYO GitHub App per tenant.** The tenant git-config optionally carries a credential *reference* — `appId`, `installationId`, optional GHE `apiBase`, and an ESO-backed secret reference for the private key. The key material itself is never stored in the database and never leaves ESO-mounted files (unchanged from ADR-0004/plan §12.2). Providers are constructed per tenant and cached by tenant; a 401/403 evicts the cached provider so the next deploy reloads the key (rotation without restart); suspension or installation deletion fails the deploy with a typed revocation error plus an audit event.
- **Resolution order:** BYO override when present, else the platform app with per-org installation discovery. The legacy `INARI_GITHUB_APP_INSTALLATION_ID` seeds the installation cache as a backwards-compatible default and is deprecated.

The orchestrator depends on a `gitprovider.Resolver` seam (`ForTenant(ctx, gitCfg)`) instead of a single injected provider; modules that address platform-owned repos directly (scaffolding, tenant zone factory) resolve per-repo via the platform app only.

## Consequences

- Tenants can hold state repos in their own GitHub orgs; one platform app serves all of them, and security-conscious tenants can fully isolate with their own app (own rate-limit bucket, own GHE host).
- Credential events (`tenant.git.auth.resolved`) and deploy audit records carry `authModel`/`installationId`/`apiBase`, so every git write is attributable per tenant and host.
- Git-config API validates BYO references (422), including https-only, `/api/v3`-only apiBase values with an optional platform host allowlist (SSRF containment).
- A suspended *platform* app takes down all model-A tenants; this blast radius is documented and surfaced via the per-tenant `gitProviderStatus` probe.
- Installation-list pagination cost at cold cache is amortized by warming the cache for all orgs from one list call; a write-through org→installation hint table is a flagged follow-up if scale demands it.
- We would revisit if GitHub introduces per-org app delegation that removes the need for tenant-side installs.
