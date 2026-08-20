# 8. Multi-org UX: global "All tenants" home with strict per-org scoping

- Status: Accepted
- Date: 2026-08-13
- Deciders: Inari platform engineering
- Source: [Inari platform plan](../architecture/inari-platform-plan.md) §11 Decision 6, applied in §8.1

## Context

Keycloak Organizations support multi-org users (e.g. contractors spanning tenants). The console had to choose between: only per-tenant views (forcing constant switching), or a global aggregate view (risking cross-tenant data bleed in the UI). Tenancy is Inari's core invariant — any aggregate UX must not weaken scoping.

## Decision

Users belonging to several tenants get a global **"All tenants" home** (aggregated resources, pending approvals, notifications across orgs), a fast switcher with recents, **strict per-org scoping once a tenant is selected**, and deep links that carry tenant context.

## Consequences

- Multi-org users get a usable landing experience without weakening the tenant boundary: every API call remains tenant-scoped and OpenFGA-checked.
- Aggregate views are read-only rollups of per-tenant queries; mutations always occur inside an explicit tenant context.
- Deep links must carry tenant context — link-sharing across org boundaries fails closed (permission denied), never leaks.
- We would revisit the aggregate model if cross-tenant dashboarding becomes a first-class requirement.
