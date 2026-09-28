# 15. Release bundle repo + standardized edge releases

- Status: Accepted
- Date: 2026-09-28
- Deciders: Inari platform engineering

## Context

[ADR-0002](0002-release-automation-release-please.md) standardized *stable* releases, but two gaps remained:

1. **Chart topology drifted from release reality.** The `inari-server` and `inari-console` charts lived in and released from their component repos, even though server and UI release in close lockstep and their charts are deployed together. Chart OCI namespaces had fragmented into three schemes (org-level `oci://ghcr.io/7k-inari/charts` for agent/operator, repo-scoped paths for server/ui/helm-charts).
2. **No standard increment between stable releases.** Only some repos published per-commit `:edge` images (inari-server, inari-agent, inari-operator), each ad-hoc: no GitHub release, no semver-coherent tag, and several consumers (gitops pins, e2e, UI contract sync) had no uniform way to reference "the latest build of the next release".

Options for the chart topology included keeping charts in component repos with central publishing via dispatch (rejected: sources stay scattered) and a full umbrella chart / central versions manifest (rejected for now: the chart set + compatibility range is enough parity surface). Options for edge versioning included last-release-as-is (sorts *before* the stable in semver tooling) and last-release+patch (loses the release-please proposal's intent); the pending Release PR version won on correctness.

## Decision

### Release bundle repo

- `inari-helm-charts` is **renamed `inari-release-bundle`** and becomes the home of the platform's core charts. The `inari-server` and `inari-console` charts **move** from their component repos into it (versions seeded at 0.1.9 / 0.3.1 for continuity; new tag scheme `<chart>-vX.Y.Z`). The `platform-config` chart is **renamed `inari-platform`** (the ArgoCD Application and helm release keep the legacy name `platform-config` so installs are not recreated).
- **All charts publish to one org-level OCI namespace: `oci://ghcr.io/7k-inari/charts/<chart>`** (agent/operator charts already publish there and stay in their repos). The publish path is hardcoded so it never embeds the repo name again. Repo-scoped namespaces are deprecated; existing artifacts stay published.
- **Version parity is appVersion-based.** After their publish pipelines succeed, inari-server and inari-ui dispatch `appversion-bump` to the bundle repo's `chart-sync.yml`, which commits `fix(<chart>): bump appVersion to vX.Y.Z` (plus inari-console's `bundle.tag`) so release-please proposes a chart patch release — charts release in lockstep with their components through the normal human-gated Release PR. The inari-operator keeps its in-repo `extra-files` appVersion sync; the inari-agent chart keeps release-time version==appVersion parity in its own repo.
- The `inari-platform` chart declares the **supported inari-agent version range** (`agent.supportedRange`, human-maintained; `agent.recommended`, auto-bumpable by the agent release pipeline), rendered into the `inari-agent-compat` ConfigMap for inari-server to consume.

### Standardized edge releases

Every inari repo except `7k-app-of-apps` cuts an **edge release on every merge to `main`** (skipped on release merges, which the stable pipeline covers):

- Version: **`<pending-release-version>-edge.<shortsha>`**, where the pending version comes from the open release-please Release PR's bumped manifest (fallback: manifest/latest stable tag + patch; unversioned repos start at 0.1.0). Resolution lives in an identical `scripts/resolve-edge-version.sh` copy per repo.
- Output: a **GitHub prerelease** (never a full release — this keeps "latest release" pointers and release-please's last-release detection on stable releases) **plus the commit's artifacts tagged with the same semver-edge tag** (images, OCI bundles/charts/artifacts, npm `edge` dist-tag, goreleaser snapshot archives; per-repo table in [docs/ops/release-process.md](../ops/release-process.md)). Moving `:edge` tags stay.

## Consequences

- **Easier:** one chart namespace to browse/mirror/secure; server+console charts version and deploy in lockstep with their components through the existing release-please gate; every merge to main is deployable/referenceable by a semver-coherent edge tag; gitops and e2e can pin exact edge builds.
- **Harder:** cross-repo dispatch needs a token (`RELEASE_PLEASE_TOKEN`) with `contents:write`/`actions:write` on the bundle repo present in inari-server/inari-ui — a deliberate, documented exception to ADR-0002's no-PAT stance; npm OIDC trusted publishers must be registered per edge workflow filename on npmjs.com; `resolve-edge-version.sh` is a deliberate identical copy in 12 repos (drift risk, documented in the ops doc); edge prerelease tags must never be treated as stable by tooling — canary validation on `inari-ui-plugin-sdk` precedes the full rollout.
- **Revisit if:** edge release volume becomes noise (consolidate); the parity surface outgrows appVersion sync (adopt a central versions manifest or release train); or release-please manifest semantics change.
