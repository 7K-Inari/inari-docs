# M1 Spike — Bundle-managed ArgoCD lifecycle

**Status:** complete · **Track B (reproducible harness + preliminary analysis)** · **Recommendation: GO-WITH-CONDITIONS**

Plan references: §5.3 (agent lifecycle, GitOps decisions), §11 decision 1 (bundle-managed by default, BYO flag), §12.1/3 (brownfield semantics), §12.2 item 8, §5.12 (tenant-zone baseline).

## Context

Decision 1 (§11) commits Inari to installing and lifecycle-managing ArgoCD inside every tenant cluster as part of the tenant-zone baseline, with a BYO flag to adopt a pre-existing installation. This makes the agent responsible for ArgoCD version upgrades across the fleet, and makes "what do we do with the ArgoCD we found?" a day-one question. This spike drills: (1) the agent-managed version-upgrade path on kind, (2) the BYO detection/adoption flow with a documented version-skew policy, and (3) the risks this creates for the M2 git-renderer work. It gates M1 because the tenant-zone baseline (and therefore every later demo and pilot) embeds this lifecycle.

## Method

**Track B.** No container runtime is available in the spike environment (unprivileged user, no docker socket — verified), so the lifecycle drill and BYO flow are delivered as reproducible scripts plus analysis grounded in upstream ArgoCD upgrade notes. Claims marked **PRELIMINARY** must be confirmed by re-running the harness; this is a stated condition of the GO.

Harness: `docs/spikes/harness/argocd/`

- `run-upgrade-drill.sh [vA vB]` — kind cluster; installs the "bundle" (upstream pinned `install.yaml` for version A, namespace `argocd`); seeds a test Application (`test-application.yaml`, guestbook with automated sync); upgrades A → B by applying the new pinned manifests; verifies the Application still reconciles and health status survives; then rolls back B → A and re-verifies. Captures `version-*.txt` / `apps-*.txt` artifacts. Default drill: `v2.14.11 → v3.0.6` (a deliberately wide jump: v2 → v3 is the harshest upgrade ArgoCD has shipped — dex removal, API behavioral changes — so it bounds the difficulty of any single-minor upgrade).
- `detect-byo.sh` — the BYO probe logic the agent will run at registration: locates `argocd-server` across namespaces, reads the deployed image version, infers install method (Helm secrets vs plain manifests), inventories existing Applications/AppProjects **read-only**, and classifies the find against the skew policy below. Exit paths: none found → bundle-managed default; found + in-skew → adoption candidate; found + out-of-skew → refuse adoption, document tenant upgrade or side-by-side install.

## Findings

**PRELIMINARY** (harness re-run required — see Recommendation):

1. **Bundle-managed upgrades are low-risk within a major line and survivable across the v2→v3 boundary.** ArgoCD's upgrade model is declarative-manifest friendly: CRDs are installed with the manifests, `Application`/`AppProject` are `argoproj.io/v1alpha1` and stable across all supported versions, and existing Applications keep reconciling through control-plane restarts (the controller resumes from etcd/API state; no desired state is lost). The v2→v3 jump adds one-time concerns: dex is no longer deployed by default (SSO config moves), and deprecated config keys are dropped — both are bundle-content concerns, not per-cluster improvisation, which is exactly why the bundle must be a versioned, catalog-tested artifact. Rollback = re-apply previous bundle; CRDs are forward-compatible within supported skew.
2. **The upgrade path must ride the fleet machinery.** An ArgoCD upgrade is a fleet-wide change with blast radius (§10): bundle versions roll out as staged rollouts — canary ClusterSet → waves, health gates from agent-reported ArgoCD/Application health, rollback-by-snapshot. The drill script is the CI gate a bundle version must pass before entering the `stable` channel.
3. **BYO detection is reliable; adoption must be conservative.** Version and install method are cheaply and reliably detectable (image tag, Helm ownership secrets, `/api/version` once credentials exist). The risky part is mutation semantics: a pre-existing ArgoCD holds tenant-owned Applications, AppProjects, and repo credentials. Per §12.1/3 the default is `observe-only`: Inari inventories the installation and manages **only** its own namespaced footprint (dedicated AppProject per tenant team, Inari-created Applications labeled for ownership). `adopt` (Inari manages the installation's lifecycle, including upgrades) is an explicit, audited, per-cluster opt-in and should be rare in v1 — adopting an install we did not lay down means inheriting unknown HA/RBAC/SSO customizations.
4. **Version-skew policy (deliverable).** Mirror the agent upgrade contract (§5.11): **the agent supports bundle-managed ArgoCD minor lines N and N−1**; bundles pin exact patch versions. BYO adoption requires the found version to be within the supported window of the bundle line the agent would have installed (detection script defaults: `SUPPORTED_MIN=v2.14.0`); older installs are refused adoption (agent runs observe-only, surfaces an "upgrade your ArgoCD" finding) — never silently upgraded. Newer-than-bundle installs are observe-only until the bundle catches up.
5. **Risks for the M2 git-renderer work** (deliverable):
   - **Resource-tracking method.** ArgoCD historically defaulted to `label` tracking, with `annotation`/`annotation+label` recommended; the bundle must pin `application.resourceTrackingMethod: annotation` from day one — a BYO instance on legacy `label` tracking behaves differently for pruning, and the renderer must not assume the bundle's method on adopted installs.
   - **AppProject tenancy.** The renderer registers Applications into per-tenant-team AppProjects; a BYO install with restrictive `AppProject` RBAC or a locked-down `default` project can reject Inari Applications. Renderer must surface this as a clear deploy-time error, not a sync loop.
   - **Health/customizations.** The bundle can rely on bundled Lua health checks and sync-wave conventions for baseline components; BYO installs may already define conflicting customizations in `argocd-cm` — Inari must merge, never overwrite, that ConfigMap on adopted installs (or scope customizations to its own Applications).
   - **API version floor.** Renderer features must target the `argoproj.io/v1alpha1` Application API only; anything newer (e.g. ApplicationSet generators beyond the bundled set) is bundle-managed-only behavior, gated by detected version.
   - **ApplicationSet.** v1 scope: Applications only; ApplicationSet is a bundle-managed convenience the renderer may use later but must not require on BYO installs.

## Recommendation

**GO-WITH-CONDITIONS.** Bundle-managed ArgoCD with a BYO flag remains the right v1 decision: upgrades are declarative and reversible, the skew policy is simple to state and enforce, and the risks to M2 rendering are concrete and mitigable. Conditions:

1. Re-run `harness/argocd/run-upgrade-drill.sh` (including rollback) on a docker-capable workstation/CI for the current bundle line before M1 exit sign-off; attach `version-*.txt`/`apps-*.txt` artifacts and remove PRELIMINARY markers.
2. The bundle is a versioned artifact in `inari-catalog`/charts with `application.resourceTrackingMethod: annotation` pinned; upgrades ship only through staged fleet rollouts with this drill as the channel-promotion CI gate.
3. BYO default is `observe-only`; `adopt` is explicit, audited, per-cluster; out-of-skew installs are never auto-upgraded.
4. The M2 git-renderer targets `argoproj.io/v1alpha1` Applications + per-team AppProjects only, and handles restrictive-RBAC BYO installs with explicit deploy-time errors.

## Impact if no-go

If the re-run shows the upgrade path is unreliable (e.g. CRD/data migration issues across supported versions), revisit §11 decision 1: **BYO-only at v1** (agent detects + adopts, never installs) is the low-cost fallback — it shrinks the tenant-zone baseline (§5.12) and weakens the "batteries included" story but keeps every other decision intact. The harsher fallback — bundle-managed but pinned to a single ArgoCD version with upgrades deferred to v1.x — trades a §10 fleet-blast-radius risk for a security-patch lag and is only acceptable with a documented patch SLA.
