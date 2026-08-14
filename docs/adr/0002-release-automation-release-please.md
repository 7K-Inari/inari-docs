# 2. Release automation via release-please (PR-only mode + merge-triggered release workflow)

- Status: Accepted
- Date: 2026-08-14
- Deciders: Inari platform engineering

## Context

Inari is a 12-repo polyrepo ([plan §6](../architecture/inari-platform-plan.md#6-repository-topology)) where every repo ships an independently versioned artifact: container images, Go modules, an npm package, goreleaser binaries, OCI Helm charts, and catalog OCI artifacts. Without a uniform process, each repo would grow bespoke tagging and publishing scripts, versions would drift from commit history, and the manual toil would scale with the repo count.

The supply-chain requirements in [plan §5.10](../architecture/inari-platform-plan.md#510-security-model-summary) — cosign-signed images/artifacts, SBOMs, and SLSA provenance on release images — must attach to every release, so the release pipeline is security-critical and must be consistent everywhere.

We also want a human gate before anything publishes: version bumps and CHANGELOGs deserve review, and releases should be an explicit maintainer action, not a side effect of merging any PR.

A key technical constraint shaped the design: tags and releases created inside a workflow via the default `GITHUB_TOKEN` do **not** fire downstream `on: push: tags:` or `on: release` workflow triggers (GitHub's anti-recursion rule). A "tag in one workflow, publish in a tag-triggered workflow" split therefore does not work with the default token.

Alternatives considered:

- **release-please full mode** (creates tags + GitHub Releases itself): conflicts with the GITHUB_TOKEN limitation — publish workflows would never fire. Using a PAT/GitHub App token to work around it adds a long-lived credential to every repo.
- **Tag-triggered publishing with a bot token**: works, but spreads a privileged PAT across 12 repos and loses the reviewed-Release-PR gate unless we build it ourselves.
- **Fully manual releases**: maximum control, maximum toil and drift; CHANGELOG quality degrades immediately.

## Decision

We will standardize all 7K-Inari repos on the following release process:

1. All commits to `main` follow Conventional Commits (squash-merged PRs with conventional titles).
2. **[release-please](https://github.com/googleapis/release-please) runs in PR-only mode** (`skip-github-release: true`). It opens and maintains a Release PR containing the version bump and CHANGELOG diff. It never creates tags, never creates GitHub Releases, and never triggers publish pipelines.
3. **A maintainer manually merges the Release PR** — this merge *is* the release gate.
4. A repo-local **`release.yml` workflow, triggered `on: push` to `main`**, detects the release merge, creates the git tag and GitHub Release, and runs the publish jobs (GHCR images + cosign + SBOM/SLSA, OCI charts/artifacts, goreleaser binaries, Go module/npm tags) directly or via `workflow_call` reusable workflows.

Publish jobs live in the same workflow that creates the tag precisely because of the GITHUB_TOKEN trigger limitation above. Per-repo release-types, the per-chart scheme for `inari-helm-charts`, and the per-package tag scheme for `inari-catalog` are specified in [docs/ops/release-process.md](../ops/release-process.md), the operational documentation of record. `inari-docs` itself ships no releases.

## Consequences

- **Easier:** every repo releases identically; versions and CHANGELOGs derive from commit history; supply-chain signing/provenance is enforced by one workflow shape; contributors learn one process (documented in the ops doc) and can cut a release in any repo; no long-lived PATs needed for the core flow.
- **Harder:** releasing requires a maintainer merge of the Release PR (deliberate — it is the gate); release-please config (`release-please-config.json`, manifest) must be maintained per repo, including per-package entries in the two monorepo-style repos (`inari-helm-charts`, `inari-catalog`); `release.yml` becomes a security-critical workflow whose changes deserve extra review.
- **Revisit if:** GitHub changes the GITHUB_TOKEN trigger semantics; the org adopts a different release tool; or we move tag creation to a GitHub App token, at which point tag-triggered publish workflows become viable and the single-workflow constraint can be relaxed.
