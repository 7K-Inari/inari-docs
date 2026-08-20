# Operator Guide

Documentation for **platform engineers and operators** running the Inari control plane and platform cluster.

## Contents

- [Day-0 bootstrap](bootstrap.md) — install the platform cluster and control plane from `inari-helm-charts`
- [Backup & restore (DR runbook)](backup-restore.md) — PostgreSQL, OpenFGA, Keycloak, NATS; tested restore procedure
- [Fleet rollouts & agent upgrades](fleet-rollouts.md) — ClusterSets, staged rollouts, drift detection, upgrade channels
- [Policy packs](policy-packs.md) — request-time OPA, render-time checks, Kyverno/CEL admission packs, exemptions
- [Tenant Zone vending](tenant-zones.md) — management-account setup, `tenant-zone-aws` flow, decommission
- [Cluster lifecycle](cluster-lifecycle.md) — states, cordon, decommission

Also see:

- [Threat model](../security/threat-model.md) — security posture per trust zone
- [Load test report](../ops/load-test-v1.md) — validated v1 scale envelope
- [Upgrade/downgrade drill](../ops/upgrade-downgrade-drill.md) and [fleet rollout game-day](../ops/fleet-rollout-gameday.md) — M4 drill notes
- [Release process](../ops/release-process.md)
