# M1 Spike — KRO `v1alpha1` upgrade drill

**Status:** complete · **Track B (reproducible harness + preliminary analysis)** · **Recommendation: GO-WITH-CONDITIONS**

Plan references: §5.5 (catalog & capability discovery, KRO as curated-package format), §10 (risk: "KRO API maturity (`v1alpha1`)"), §12.2 item 7.

## Context

Inari's curated catalog is built on KRO `ResourceGraphDefinition`s (`v1alpha1`): RGDs are single-YAML, CEL type-checked packages from which kro generates a CRD + controller dynamically on tenant clusters. The §10 risk is explicit: `v1alpha1` may break. The mitigation the plan commits to is (a) isolating kro behind the `CatalogItem` abstraction, (b) pinning kro versions, and (c) a raw-CRD fallback path. This spike drills all three: simulate an RGD API break on a kind cluster, verify tenants are shielded, prove the fallback, and produce version-pinning guidance.

## Method

**Track B.** The execution environment for this spike has no container runtime (no docker/kind possible — verified: unprivileged user, no docker socket), so the drill is delivered as a fully scripted, reproducible harness plus analysis grounded in kro's upstream release behavior. Every quantitative claim below is **PRELIMINARY** until the harness is re-run; re-running is a stated condition of the GO.

Harness: `docs/spikes/harness/kro/`

- `run.sh` — creates a kind cluster, installs kro at version A (Helm, OCI chart `oci://ghcr.io/kro-run/kro/kro`), applies the RGDs and sample instances, snapshots all RGDs/CRDs, **upgrades kro A → B (the simulated API break)**, snapshots again, diffs, verifies existing instances still reconcile, then exercises the raw-CRD fallback. Run: `./run.sh 0.2.3 0.3.0`.
- `rgd-namespace-as-a-service.yaml`, `rgd-web-service.yaml` — representative RGDs shaped like planned `inari-catalog` packages.
- `instances.yaml` — tenant-facing instances (what a deploy wizard would emit).
- `fallback-raw-crd-instance.yaml` — the same composed resources as plain manifests, bypassing the RGD entirely.

The drill's break scenarios, in increasing severity:

1. **kro controller upgrade with unchanged RGD API** — the common case; expect zero tenant impact.
2. **RGD schema change (`v1alpha1` revision)** — e.g. restructured `spec.schema` fields; existing RGD objects may fail re-validation or the generated CRD shape changes.
3. **kro behavioral break** — generated CRD group/version renamed, status fields moved, or instance CRD conversion required.

## Findings

**PRELIMINARY** (harness re-run required — see Recommendation):

1. **Where the break lands.** An RGD API break surfaces in exactly three places: the `ResourceGraphDefinition` CRD itself, the *generated* CRDs (`<kind>.kro.run`) and their stored instances, and the CEL expressions in RGD templates. Tenant-facing deploy specs (instances of generated CRDs) are kro-implementation-detail objects — in Inari they are rendered by the orchestrator from `CatalogItem` schemas, so tenants never author them by hand. This is the core shielding argument: **a breaking RGD change is absorbed at the catalog layer by re-deriving the `CatalogItem`'s OpenAPI schema and re-rendering instances**, provided the catalog service treats RGDs as a build-time input, not a runtime API tenants depend on.
2. **What leaks through the abstraction.** The generated CRD's `group/kind/version` appears in tenant Git (rendered instance manifests in `<tenant>-inari-state` repos) and in ArgoCD Applications. If a kro upgrade renames or re-versions generated CRDs, rendered manifests in tenant repos must be rewritten — an orchestrator migration job, not a tenant task, but a real operational cost that must be rehearsed. Instance `status` shape changes leak into the Resources Inventory health mapping.
3. **Existing instances survive kro upgrades in the normal case** — upstream kro treats generated CRDs and their instances as durable across controller upgrades; the dangerous cases are (a) CRD schema tightening that invalidates stored instances (requires conversion or re-apply) and (b) RGD deletion semantics (deleting an RGD cascades to generated CRD and instances — must be guarded by admission policy; this belongs in the baseline policy packs, §5.11).
4. **Raw-CRD fallback is structurally sound.** Every RGD composes ordinary Kubernetes resources; the fallback is the orchestrator rendering those same resources directly into tenant Git (`fallback-raw-crd-instance.yaml` demonstrates this shape for the drill's packages). Because Inari's execution path is GitOps (render → tenant Git → ArgoCD), bypassing kro does not change the delivery mechanism — only the templating layer. Capability discovery still inventories the resulting CRDs/resources, so the catalog does not rot when a package drops from RGD to raw.
5. **kro's EKS-managed-capability trajectory** (tracking note from §10): if kro becomes an EKS-managed add-on, version pinning partially moves to AWS's channel model — another argument for pinning guidance that tolerates multiple kro versions across the fleet.

**Version-pinning guidance (deliverable):**

- Pin kro per cluster via the tenant-zone baseline; record `kroVersion` on the Cluster record via capability discovery.
- Every curated `CatalogItem` declares `requiresKro: ">=x.y <x.(y+1)"`-style compatibility, checked at render time and in catalog CI.
- kro upgrades are fleet rollouts (§5.11): canary ClusterSet → waves, health-gated, rollback-by-snapshot; never ad-hoc per cluster.
- Catalog CI runs this drill (`run.sh`) on every kro release and every RGD change — contract testing for RGD schema drift.
- Admission guard (Kyverno/CEL VAP in the baseline pack): block deletion of `ResourceGraphDefinition`s that have live instances.

## Recommendation

**GO-WITH-CONDITIONS.** The `CatalogItem` abstraction shields tenants by construction (RGDs are a build-time input; tenants consume OpenAPI schemas and rendered manifests), and the raw-CRD fallback preserves the GitOps delivery path end-to-end. Conditions:

1. Re-run `harness/kro/run.sh` on a docker-capable workstation/CI against the two most recent kro minor versions before M1 exit sign-off; attach the `crd-diff.txt` and events artifacts to this report (replace PRELIMINARY markers).
2. `requiresKro` compatibility metadata on curated catalog items + render-time check — implement with the catalog service in M2.
3. kro upgrades ride the staged-rollout machinery (§5.11) with the drill as catalog CI.
4. Admission guard against RGD deletion cascades in the baseline policy pack.

## Impact if no-go

If the re-run shows existing instances do **not** survive a kro minor upgrade, or the generated-CRD shape churns every release: keep the `CatalogItem` model but demote kro from the golden-path format — render RGDs to static manifests in catalog CI (kro as a build tool only, never installed on tenant clusters), or fall back to Helm for curated packages in v1 and revisit kro at its `v1beta1`. This would edit §5.5 (packaging decision), §7.1 (curated catalog v1 format), and the M2 scope (RGD render pipeline becomes Helm/static render).
