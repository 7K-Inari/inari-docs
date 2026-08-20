# Tenant Zone vending

How to connect the AWS management account and vend new Tenant Zones — a new AWS organization account + provisioned tenant cluster + auto-installed baseline, in one governed flow (plan §5.12). v1 scope: AWS-only, EKS-only, new accounts under an existing AWS Organization, single **starter tier** ([ADR-0009](../adr/0009-tenant-zone-single-starter-tier.md)).

## Prerequisite: connect the management account

The platform cluster must be connected to the AWS Organizations **management account** as a special `CloudAccount` (`scope: management`) whose role allows **only**:

- `organizations:CreateAccount`, `organizations:TagResource`, `organizations:DescribeCreateAccountStatus`
- account-closure actions (for decommission)

1. In the console: **Cloud Accounts → Register AWS account**, choose **Management account** scope.
2. Apply the generated CloudFormation/Terraform snippet in the management account — it creates a role trusting the platform cluster's OIDC provider (`AssumeRoleWithWebIdentity`, conditioned on `sub`/`aud`).
3. Validation runs as a dry-run assume-role. Least-privilege: nothing beyond the actions above.
4. **Request the minimum AWS account-creation quota** you need to start — additional quota with demand.

## Vending a zone

Platform engineers only; approval- and policy-gated by default. Console: **Tenant Zones → Vend new zone** (or the `tenant-zone-aws` catalog item).

1. **Request** — name/slug, OU placement, region, cluster tier (starter). The Policy Service pre-flights naming rules, OU quota, budget guardrails; default policy requires approval.
2. **Account vend** — Crossplane (`provider-aws-organizations`) on the platform cluster creates the account in the target OU and waits for readiness (minutes). A baseline policy pack applies org-side guardrails (CloudTrail, budget alert, mandatory cost tags).
3. **Trust bootstrap** — via `OrganizationAccountAccessRole`, the standard OIDC web-identity role is created in the new account (same contract as BYO onboarding). From here everything is `AssumeRoleWithWebIdentity` — no stored keys.
4. **Cluster provision** — Crossplane EKS resources (cluster, node group, OIDC provider for IRSA) per the starter tier.
5. **Inari wiring** — Keycloak Organization + default groups and RBAC mapping; Cluster + CloudAccount records; registration token; the tenant-zone baseline bundle (inari-agent, ArgoCD, ESO, baseline policy packs) rendered into the zone's new Git repo with the ArgoCD root app registered. The agent connects, exchanges the token, and the bootstrap credential is forgotten.
6. **Active** — capabilities start streaming; the curated catalog becomes visible per tenant policy.

The flow is **resumable and idempotent** — long-running AWS operations are tracked as sub-resources with status. A zone stuck on an AWS-side failure exposes a manual-intervention path in the zone detail view (plan §10).

## Decommissioning a zone

Reverses the flow, behind approval gates with ownership checks: **cordon → drain Inari-managed resources → delete EKS → CloseAccount/suspend → revoke identities → archive audit**. See [cluster lifecycle](cluster-lifecycle.md) for the drain semantics.

## Guardrails

- Org/OU quota pre-checks, region allow-lists, mandatory cost tags enforced at request time.
- Every step emits audit events.
- Zones carry a `TenantZone` record linking org, cloud account, cluster, tier, state, and provisioner.

## Step-by-step walkthrough

See the tutorial: [Vend your first Tenant Zone](../tutorials/vend-a-tenant-zone.md).
