# 9. Tenant Zone Factory: single starter tier at v1

- Status: Accepted
- Date: 2026-08-13
- Deciders: Inari platform engineering
- Source: [Inari platform plan](../architecture/inari-platform-plan.md) §11 Decision 7, applied in §5.12

## Context

Tenant Zone vending could launch with multiple cluster tiers (sizes, regions, distros) or a single opinionated tier. Each tier multiplies the provisioning matrix (Crossplane compositions, quota pre-checks, policy packs) and the AWS account-creation quota Inari must request. AWS Organizations `CreateAccount` quotas are the hard external constraint (plan §10).

## Decision

v1 ships a single **starter tier**: one region, EKS with a minimal managed node group (e.g. 2× t3.large). Inari requests only the **minimum AWS account-creation quota** needed to start. Additional tiers land with demand. v1 scope remains AWS-only, EKS-only, new accounts under an existing AWS Organization.

## Consequences

- The vending flow is exercised end-to-end on one well-tested path; quota asks stay small and approvable.
- Tenants needing larger clusters provision additional capacity in-zone after vending, or wait for v1.x tiers.
- Import of existing org accounts, Control Tower/AFT coexistence, other distros, and Azure/GCP zones are explicitly deferred (plan §5.12).
- We would add tiers when pilot demand demonstrates the need — tracked as a v1.x roadmap item.
