# 13. Per-user git connections (model C)

- Status: Proposed
- Date: 2026-09-26
- Deciders: Inari platform engineering
- Source: [Inari platform plan](../architecture/inari-platform-plan.md) §5.3 (GitOps/git auth); extends [ADR-0004](0004-platform-owned-tenant-state-repos.md) and [ADR-0010](0010-hybrid-github-app-model.md)

## Context

ADR-0010 established a hybrid, two-model git credential architecture — **model A** (one platform GitHub App with per-tenant installations) and **model B** (BYO GitHub App per tenant). Both models are **tenant-scoped**: every commit, PR, and repo operation is performed by an app/bot identity shared by the whole tenant. That is correct for state-repo automation (ADR-0004), but it falls short for user-initiated scaffolding flows:

- **Attribution**: commits appear as the app, not the developer who clicked "create" — weak audit and a poor code-review trail.
- **Repo ownership**: developers cannot scaffold new application repos into their own account or org; everything funnels into platform- or tenant-owned locations.
- **Least privilege**: any tenant member implicitly wields tenant-level git credentials, even for personal experiments.

Alternatives considered:

1. **Per-user PAT entry.** Rejected: poor UX, manual rotation burden, unscoped storage risk — exactly the credential sprawl ADR-0004/§5.3 moved away from.
2. **Impersonation via the platform app.** Rejected: GitHub Apps cannot impersonate arbitrary users; commits would still be app-authored.
3. **Per-user OAuth connection via the GitHub App user OAuth flow (chosen, "model C").** Users authorize the app once; the control plane holds expiring user-to-server tokens plus refresh tokens and acts *as the user* for user-initiated flows, with real authorship and per-user revocation.

## Decision

We will introduce a **third auth model — model C: per-user git connections** — alongside ADR-0010's models A/B, centered on a `UserGitConnection` abstraction.

### UserGitConnection abstraction

- A **`UserGitConnection`** binds (user × tenant × git provider) and stores the OAuth material needed to act as that user. It is created through an explicit "Connect git account" flow in the console/CLI, is visible and revocable by the user, and deletion/revocation is audited.
- **GitHub first**, using the **GitHub App user OAuth flow**: short-lived, expiring **user-to-server access tokens** plus **refresh tokens** (expiring-user-token mode enabled on the app).
- **GitLab and Forgejo are planned** behind the same provider interface; the `UserGitConnection` record and resolver seam are provider-agnostic from day one (mirroring ADR-0010's `apiBase`/GHE handling).

### Model C in the gitprovider resolver

- The ADR-0010 `gitprovider.Resolver` seam is extended with **`AuthModelUser`** and a **`ForUser(ctx, userID, gitCfg)`** resolution path, next to the existing `ForTenant(ctx, gitCfg)`.
- **Resolution order**: BYO override (model B) → platform app (model A) → user connection (model C). Call sites that require user attribution (scaffolding, user-initiated commits) request `ForUser` explicitly; automation paths (state-repo reconciliation) keep using `ForTenant` and are unchanged.
- **Commit authorship**: commits and PRs made on behalf of a user carry the **connected user as author and committer**, so history and review trails show the real person.
- **Real repo creation in `EnsureRepo`**: when scaffolding in user scope, `EnsureRepo` creates the repository in the **user's own account/org** through their connection, rather than defaulting to the platform-owned `<tenant>-inari-state` repo (ADR-0004 state repos remain platform-owned and unaffected).

### Template manifest: `scaffold.scope`

- The template (catalog item) manifest gains **`scaffold.scope: platform | user`**, defaulting to **`platform`** — existing templates are unaffected.
- `scope: user` declares that scaffolding acts as the requesting user: repo created in the user's account/org, commits authored by the user. It **requires an active `UserGitConnection`** for the requesting user.

### Tenant fallback policy: `user_template_fallback`

- A per-tenant policy, **`user_template_fallback`**, governs what happens when a `scope: user` template is requested by a user with **no active connection**:
  - **`block` (default)** — the request fails with an actionable error prompting the user to connect their git account.
  - **`platform_app`** — the request falls back to the tenant's platform-app credentials (model A), clearly attributed as a fallback in the audit record.
- The policy is **admin-only** to change and every change is **audited**; fallback events themselves are also audited (`tenant.git.auth.resolved` gains `authModel: user | platform_app` and `fallback: true` fields, consistent with ADR-0010).

### Refresh-token protection

- Refresh tokens are **envelope-encrypted** at rest: a **Vault/OpenBao transit key** acts as the KEK, wrapping per-connection **AES-GCM** data keys. Plaintext refresh tokens never touch the database, backups, or logs.
- **Single-flight refresh**: at most one in-flight token refresh per connection; concurrent requests share the result, preventing refresh storms and accidental token rotation races.
- **Refresh-token reuse detection**: because GitHub rotates refresh tokens on use, a presented-but-already-rotated token signals possible theft. On reuse detection the connection is **immediately revoked**, the user is prompted to reconnect, and a **security audit event** is emitted.

## Consequences

- **Easier**: real per-user attribution in git history and PRs; developers can scaffold repos into their own accounts/orgs; per-user revocation shrinks blast radius versus tenant-wide app credentials; the provider interface sets up GitLab and Forgejo without redesign.
- **Harder**: a user-token lifecycle to operate (expiry, refresh, revocation, reconnect UX); encryption key management via Vault/OpenBao transit becomes a hard dependency of the control plane; the `user_template_fallback` policy adds a governance surface that must stay admin-only and audited; `EnsureRepo` gains a second, user-scoped creation path to test.
- **Phased implementation**:
  - **Phase 1** — GitHub user OAuth flow, `UserGitConnection` storage + envelope encryption, resolver `AuthModelUser`/`ForUser`, `scaffold.scope`, `user_template_fallback` policy, single-flight refresh + reuse detection.
  - **Phase 2** — GitLab provider behind the same interface.
  - **Phase 3** — Forgejo provider behind the same interface.
- **File-level impacts** (product repos, for the implementation waves): gitprovider resolver and provider interface; orchestrator `EnsureRepo`/scaffolding path; template manifest schema + validation; tenant git-config API (fallback policy, admin-only guard); control-plane crypto module (transit envelope encryption); audit event schemas; console/CLI connect-disconnect UX. No product code is changed by this ADR.
- **We would revisit if**: GitHub changes fine-grained user-to-server token semantics in a way that breaks the refresh model, or demand emerges for per-org *user* apps that would supersede a single platform app's user flow.
