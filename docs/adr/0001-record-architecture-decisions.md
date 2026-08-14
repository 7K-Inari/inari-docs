# 1. Record architecture decisions

- Status: Accepted
- Date: 2026-08-13
- Deciders: Inari platform engineering

## Context

Inari is a multi-tenant Internal Developer Platform with many significant, hard-to-reverse technical choices (identity fabric, agent protocol, authorization model, packaging format). These choices need to be legible to current and future contributors: what was decided, why, and what would make us revisit it. The canonical plan document records the current state but is not a decision log.

## Decision

We will keep a lightweight log of **Architecture Decision Records (ADRs)** in `docs/adr/`, rendered on this site under the "ADRs" sidebar section.

- One file per decision, numbered sequentially: `NNNN-short-title.md`.
- Format: the template in [0000-adr-template.md](0000-adr-template.md) (Status, Context, Decision, Consequences).
- ADRs are immutable once Accepted; a changed decision is a new ADR that supersedes the old one.
- Write an ADR for significant or hard-to-reverse decisions — technology choices, protocols, tenancy/identity model, public contracts, repo-topology changes. See `CONTRIBUTING.md` for details.

## Consequences

- Decisions and their rationale are discoverable in version control and on the docs site.
- Slight overhead per significant decision (one short markdown file + PR).
- The plan document remains the architecture *description*; ADRs record the *decisions* — they complement, not duplicate.
