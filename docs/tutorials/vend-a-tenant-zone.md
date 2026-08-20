# Tutorial: Vend your first Tenant Zone

Create a brand-new Tenant Zone — AWS organization account + EKS tenant cluster + auto-installed baseline — from one console request. ~45 minutes (most of it AWS-side waiting).

**Prerequisites:** platform-engineer role; the **management account connected** ([operator guide](../operator-guide/tenant-zones.md) — this is a one-time setup); account-creation quota available in your AWS Organization; approval rights if your policy requires it (default: it does).

## 1. Open the zone wizard

**Tenant Zones → Vend new zone**. Fill the form:

- Name/slug (validated against naming policy)
- OU placement in your AWS Organization
- Region and tier — v1 ships the **starter tier** (one region; EKS with a minimal managed node group, e.g. 2× t3.large — [ADR-0009](../adr/0009-tenant-zone-single-starter-tier.md))

The Policy Service pre-flights: naming rules, OU quota, budget guardrails. Fix anything it flags — denials include remediation.

## 2. Approve

With the default policy, the request lands in **Approvals**. Approve it (or have a second platform engineer approve — self-approval can be policy-blocked).

## 3. Watch the provisioning flow

The zone detail page tracks each step as a sub-resource with live status; every step is audited:

1. **Account vend** — Crossplane creates the AWS account in the target OU (`organizations:CreateAccount`; async, minutes). A baseline policy pack applies CloudTrail, budget alert, and mandatory cost tags org-side.
2. **Trust bootstrap** — the standard OIDC web-identity role is created in the new account via `OrganizationAccountAccessRole`. No stored keys anywhere.
3. **Cluster provision** — EKS cluster, managed node group, and OIDC provider for IRSA per the starter tier.
4. **Inari wiring** — Keycloak Organization + default groups + RBAC mapping; Cluster and CloudAccount records; registration token; baseline bundle (inari-agent, ArgoCD, ESO, baseline policy packs) rendered into the zone's new Git repo; ArgoCD root app registered.
5. **Active** — the agent connects, exchanges the token, capabilities start streaming.

The flow is resumable and idempotent: if an AWS-side step stalls (quota, throttling), the zone shows the stuck sub-resource and a manual-intervention path — resolve and resume rather than restarting.

## 4. Verify the new zone

- **Clusters** shows the new cluster `Active` with capabilities streaming.
- **Catalog** filtered to the zone shows the curated items your visibility policy grants new tenants.
- **Settings → Tenant/Org** shows the new Keycloak Organization; invite its first members.

## 5. (Drill) Decommission it

If this was a drill, decommission from the zone detail: cordon → drain Inari-managed resources → delete EKS → `CloseAccount`/suspend → revoke identities → archive audit, behind approval gates. Watch each step reverse cleanly — this path is as important as the create path (plan §10, AWS quota risk).

**Next:** onboard developers to the zone with the [user guide](../user-guide/index.md).
