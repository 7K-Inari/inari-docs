# 3. Bundle-managed tenant GitOps with a BYO flag

- Status: Accepted
- Date: 2026-08-13
- Deciders: Inari platform engineering
- Source: [Inari platform plan](../architecture/inari-platform-plan.md) §11 Decision 1, applied in §5.3

## Context

Every tenant cluster needs a GitOps engine for Inari-managed workloads. Two options: adopt whatever ArgoCD (or other engine) the tenant already runs ("bring your own"), or have Inari install and lifecycle-manage ArgoCD as part of the tenant-zone baseline ("bundle-managed"). BYO maximizes tenant flexibility but makes version skew, capability detection, and upgrade paths unbounded; pure bundling alienates tenants with an existing, customized ArgoCD.

## Decision

Tenant-local GitOps is **bundle-managed by default**: the agent installs and lifecycle-manages ArgoCD as part of the tenant-zone baseline. A **BYO flag** adopts an existing ArgoCD installation, with a documented version-skew policy applying to adopted installations.

## Consequences

- The common path is uniform: upgrade drills, health reporting, and action proxying target a known ArgoCD lifecycle (validated in the [M1 ArgoCD bundle-lifecycle spike](../spikes/m1-argocd-bundle-lifecycle.md)).
- BYO adoption carries an explicit skew policy; tenants on BYO own their ArgoCD upgrades.
- We would revisit if tenant demand shows a majority of clusters with pre-existing ArgoCD that cannot be adopted cleanly.
