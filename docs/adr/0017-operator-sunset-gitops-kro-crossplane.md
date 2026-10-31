# 17. inari-operator sunset: platform resources via server-rendered GitOps, KRO, and Crossplane

- Status: Accepted
- Date: 2026-10-31
- Deciders: Inari platform engineering
- Source: operator future-role assessment (Kandev task `b6969b1e`, decisions D1–D7); supersedes `inari-operator` repo `docs/decisions/0001-keycloak-admin-rest.md` (Keycloak Admin REST) and the "Follow-up (2026-09): Dex delivery via the official dexidp chart" section of [ADR-0012](0012-extension-oidc-passthrough-token-exchange.md) (ClusterDex)

## Context

`inari-operator` carried eight `platform.inari.io/v1alpha1` CRDs in three groups: thin facades over infrastructure that became first-class addons (`CertIssuer`, `DNSZone`/`DNSRecord`, `ArgoProject` over cert-manager/ExternalDNS/ArgoCD), stable platform plumbing (`TenantNamespace`), and two genuinely non-trivial reconcilers (`KeycloakRealm`/`KeycloakClient` with in-process Keycloak Admin REST idempotency/teardown, and `ClusterDex` per-cluster Dex/ArgoCD-RBAC fan-out). The multi-repo GitOps feature (first-class addons, the source-agnostic renderer in `inari-server/internal/renderer`, per-tenant state repos) removed the reason for the facades, and ArgoCD already does install/upgrade ordering (sync waves, health gating, ServerSideApply) better than a bespoke operator could. Maintaining the operator — release machinery, envtest fixtures, chart sync — cost more than the thin render logic it owned.

This ADR records the resolved assessment decisions **D1–D7**:

- **D1 — Crossplane and KRO adopted platform-wide.** Crossplane is the go-to for self-service resources (Keycloak realm/client, etc.); KRO hosts managed stacks (e.g. monitoring).
- **D2 — Timeline.** Hub gitops prerequisites (Phase 0) could start immediately; the migration phases were gated on the multi-repo GitOps feature's Phase 3 (addons + renderer live); the operator was maintained until cutover completed.
- **D3 — Repo-level authorization replaces impersonation.** `impersonateServiceAccount` is dropped. Tenant isolation = team-scoped repos + commit policy/PR review + ArgoCD AppProject destination restrictions; the server is the only repo writer. Hub-side self-service resources (Keycloak instances, OIDC clients, Vault groups) always go to the hub-facing platform repo under tenant-scoped paths (`tenants/<org>/…`), applied by hub ArgoCD — never to team repos.
- **D4 — Repo fate.** `inari-operator` is archived at the end of the sunset (Phase 4). A stripped keycloak-only controller was a temporary contingency only, never the end-state.
- **D5 — Dex fan-out targeting.** Cluster selectors are Inari inventory labels/tags evaluated server-side; the renderer emits engine-specific fan-out (ArgoCD ApplicationSet list generator today; Flux/Fleet possible later without selector changes); outbox events on cluster register/label-change trigger re-render + commit.
- **D6 — TenantNamespace becomes a KRO RGD.** KRO is already platform-wide (D1), so marginal cost is ~0; a single instance CR is a uniform server render target with a single `Ready` condition for the inventory/agent contract; lifecycle via KRO owner refs replaces the operator's finalizer machinery.
- **D7 — No backwards compatibility (pre-release).** No adoption/migration of existing CRs, no byte-identical render requirements, no dual-run feature flags, no Orphan-first Keycloak import, no rollback window. Existing dev/staging resources are deleted and recreated declaratively.

## Decision

We will **shrink and sunset `inari-operator`; the repo is archived** (D4). Each capability moved to the tool strictly better at it:

- cert-manager, ExternalDNS, and ArgoCD are **first-class addons** (`cluster_addons`); the facade CRs are replaced by **native third-party resources** (`ClusterIssuer`, `DNSEndpoint`, `AppProject`) rendered by the server straight into GitOps state repos.
- `TenantNamespace` → **KRO RGD `tenantnamespace`** (D6), versioned in the release-bundle gitops (`charts/inari-platform-graphs`); instances render as `TenantNamespace.stacks.inari.io/v1alpha1`.
- `KeycloakRealm`/`KeycloakClient` → **Crossplane `provider-keycloak`** managed resources on the hub (D1, D3), superseding the in-process Admin REST client of `inari-operator` ADR-0001. Claims (`TenantRealm`/`TenantClient` namespaced XRs, `identity.inari.io/v1alpha1`) are server-rendered to tenant-scoped paths in the hub-facing platform repo; client credentials land in tenant namespaces via ESO.
- `ClusterDex` → **server-rendered ApplicationSet pair** (dex + argocd-config) with a list generator driven by inventory selectors (D5), superseding the ADR-0012 follow-up; per-cluster OIDC client secrets stay on the server's registration seam, delivered via ESO/Vault (`configSecret.create: false` — the ADR-0012 security posture is unchanged).
- The `platform.inari.io` CRDs, the operator's ArgoCD Applications (`inari-operator`, `inari-operator-crds`), and the server-side legacy render path (including the `platform_resources.gitops_render` dual-run flag) are **removed**; remaining dev/staging CRs are deleted directly, not adopted (D7).

## Consequences

- **Easier:** no second desired-state engine competing with ArgoCD; prerequisite installs (cert-manager, ExternalDNS, ESO, KRO, Crossplane, provider-keycloak) are gitops-managed on the hub, closing the unmanaged-prerequisite gap; XRD/RGD lifecycles are versioned in git (reviewable diffs, rollback, promotion); one render path (no dual-run flag); status flows through the agent's configurable GVR watches instead of a hardcoded `platform.inari.io` group.
- **Harder / trade-offs:** confused-deputy protection now rests on server-side authorization + commit policy + AppProject restrictions rather than in-cluster impersonation (D3 accepted trade-off — a server authz bug has wider blast radius; mitigated by per-endpoint authz tests and PR policy on sensitive platform-repo paths); Keycloak realm recreation in dev/staging briefly breaks tenant-app SSO (accepted, D7).
- **Superseded:** `inari-operator` repo `docs/decisions/0001-keycloak-admin-rest.md` (direct Admin REST client → provider-keycloak); the Dex-delivery follow-up section of [ADR-0012](0012-extension-oidc-passthrough-token-exchange.md) (ClusterDex CR → server-rendered ApplicationSet). The body of ADR-0012 (token exchange) is unaffected.
- **Follow-up work:** GitHub archive of `inari-operator` (performed by the repo admins); baseline-repo `ClusterIssuer` content ownership; agent chart RBAC for the new watch groups.
- **Related:** [ADR-0004](0004-platform-owned-tenant-state-repos.md) (state repos), [ADR-0010](0010-hybrid-github-app-model.md) (git credentials), [ADR-0013](0013-per-user-git-connections-model-c.md), [ADR-0016](0016-runtime-feature-flags-openfeature.md) (flag mechanism the removed dual-run lever used).
