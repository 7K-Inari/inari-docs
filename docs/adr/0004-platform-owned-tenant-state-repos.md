# 4. Platform-owned tenant state repositories

- Status: Accepted
- Date: 2026-08-13
- Deciders: Inari platform engineering
- Source: [Inari platform plan](../architecture/inari-platform-plan.md) §11 Decision 2, applied in §5.3

## Context

Rendered resource instances must land in Git for tenant-local ArgoCD to apply. Options: commit into each tenant's application repositories, or maintain a dedicated state repository per tenant. Application repos are tenant-owned, heterogeneously structured, and often protected; writing platform-generated manifests into them couples release cadence and review process to Inari's rendering.

## Decision

Rendered instances live in a **platform-owned `<tenant>-inari-state` repository per tenant**. Application repositories stay untouched. Whether changes land via pull request or direct commit remains a per-tenant policy.

## Consequences

- Inari has a single, well-known Git target per tenant; ownership boundaries are unambiguous and teardown does not touch application code.
- One more repository per tenant to provision and govern; GitHub App credentials are delivered via ESO (never PATs).
- GitHub is the v1 provider; GitLab lands in v1.x behind the orchestrator's provider abstraction.
- We would revisit if tenants demand in-repo rendering for review-workflow reasons.
