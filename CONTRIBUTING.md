# Contributing to inari-docs

All site content lives in `docs/` (markdown; Docusaurus docs root). Edit files, open a PR — CI builds the site to verify.

## When to write an ADR

Write an **Architecture Decision Record** for decisions that are **significant or hard to reverse**, e.g.:

- Technology or library choices (language, framework, protocol, data store)
- Public contracts: APIs, agent protocol, extension SDKs
- Anything touching the tenancy or identity model
- Repo-topology or deployment-model changes
- Security posture changes

Do **not** write ADRs for routine changes: bug fixes, small refactors, docs-only updates, or decisions already recorded in the canonical plan.

### How to add one

1. Copy `docs/adr/0000-adr-template.md` to `docs/adr/NNNN-short-title.md` using the next sequential number.
2. Fill in Status, Date, Deciders, Context, Decision, Consequences.
3. Open a PR. The ADR is `Accepted` when the PR merges.
4. Once accepted, ADRs are immutable — a changed decision gets a *new* ADR that supersedes the old one.

See [docs/adr/0001-record-architecture-decisions.md](docs/adr/0001-record-architecture-decisions.md) for the decision that established this process.
