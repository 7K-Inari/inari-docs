# Policy packs & policy service

One policy model, three enforcement points (plan §5.11). The Policy Service unifies catalog visibility, parameter, approval, and git policies under a single surface — one policy surface, not four.

## Enforcement points

1. **Request-time (pre-flight)** — OPA evaluates every catalog deploy/update request against tenant + cluster policies: allowed registries, required labels, size ceilings, cost guardrails, approval requirements. Developers see the *reason* and remediation guidance, not just a denial.
2. **Render-time** — policy checks on rendered manifests in the orchestrator pipeline (block or warn) before anything reaches tenant Git.
3. **In-cluster admission** — versioned **policy packs** (Kyverno policies or CEL ValidatingAdmissionPolicies) distributed to ClusterSets via [fleet rollout](fleet-rollouts.md), sourced from `inari-catalog` (e.g. `baseline-security`, `cost-guardrails`).

Because packs and admission webhooks are agent-discovered capabilities, compliance is observable: the agent reports admission status upstream, feeding a per-cluster/per-tenant **compliance view** (Settings → Policies → Compliance).

## Operating policy packs

1. **Author/consume** — packs ship as signed OCI artifacts from `inari-catalog` with OPA tests; custom packs follow the same format (`engine: kyverno | cel-vap`, `parameters`).
2. **Assign** — a `PolicyAssignment` binds a pack version to a ClusterSet or tenant, with parameters.
3. **Distribute** — assignment triggers a staged rollout; treat admission policy changes like any fleet-wide change (canary first, health gates).
4. **Monitor** — compliance view shows per-cluster pack status; drift events flag removed/degraded admission webhooks.

## Exemptions

Exemptions are **time-boxed, approval-gated, and fully audited**:

- Created per policy, per scope (cluster, tenant, or resource), with a reason and `expiresAt`.
- Require approval per the item's approval policy.
- Expiry is enforced — there are no permanent exemptions; expired exemptions re-activate the policy.
- Every exemption and its lifecycle appears in the audit log and compliance view.

## Writing request-time policies

Request-time policies are Rego evaluated by the embedded OPA. A minimal example denying public S3 buckets:

```rego
package inari.catalog

deny contains msg if {
  input.catalogItem.name == "s3-backed-app"
  input.spec.publicAccess == true
  msg := "public S3 access is disabled platform-wide; use private access with an IRSA-scoped role"
}
```

Policies are versioned with the platform configuration and tested in CI before rollout.

## v1 scope

ClusterSets & label targeting; staged rollout distribution; request-time OPA + render-time checks; compliance view (S-priority polish). Deferred: drift auto-remediation, cost-based policies, multi-language policy authoring (plan §5.11).
