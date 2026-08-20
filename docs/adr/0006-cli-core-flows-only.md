# 6. CLI covers core flows only — no console parity in v1

- Status: Accepted
- Date: 2026-08-13
- Deciders: Inari platform engineering
- Source: [Inari platform plan](../architecture/inari-platform-plan.md) §11 Decision 4, applied in §7.2

## Context

The `inari` CLI could aim for full parity with the web console, or deliberately cover a subset. Full parity doubles the surface area of every feature (approvals, RBAC mapping UX, fleet dashboards, audit views) and competes with schema-driven console forms that do not translate well to a terminal.

## Decision

Full CLI/console parity is **not a v1 goal**. The CLI covers core flows only: `login` (OIDC device flow), cluster register, catalog browse, deploy, and resources inventory.

## Consequences

- CLI stays shippable at v1 with scripted/automation-friendly coverage of the golden path.
- Console-only features (approvals inbox, RBAC matrix, fleet views) are explicitly out of CLI scope for v1; users are directed to the console.
- Parity gaps can be added incrementally in v1.x based on usage; expanding the CLI does not require a breaking change.
- We would revisit if automation-heavy users (CI pipelines) need flows beyond the core set.
