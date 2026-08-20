# 5. Fully open source (Apache-2.0) with a services business model

- Status: Accepted
- Date: 2026-08-13
- Deciders: Inari platform engineering
- Source: [Inari platform plan](../architecture/inari-platform-plan.md) §11 Decision 3, applied in §1

## Context

Inari's commercial model had to be settled before public launch because it shapes licensing headers, repo visibility, feature gating, and community expectations. The open-core model (paid feature gating) creates perverse incentives around the extension SDK and multi-tenancy features that are Inari's differentiators.

## Decision

Inari is **fully open source under Apache-2.0** — no open-core feature gating; every capability ships in the open repositories. The commercial model is **services** (consulting, support, managed operations), not product licensing.

## Consequences

- No license-check or gating machinery in the codebase; simpler distribution and contribution story.
- Revenue depends entirely on services; the pen-test-before-services gate (see the [threat model](../security/threat-model.md)) becomes commercially critical.
- Community hygiene (trademark, DCO/CLA, SECURITY.md, supported-versions policy) is required before public launch (plan §12.3/13).
- We would revisit only if the services model proves unsustainable and a licensing change is proposed — that would be a new ADR superseding this one.
