# Fleet rollouts & agent upgrades

How to run fleet-wide change safely: ClusterSets, staged rollouts, drift detection, and the agent upgrade contract (plan §5.11). Executed **credential-free** — each stage hands desired state to the target clusters' agents, which apply it via tenant-local GitOps.

## ClusterSets

Registered clusters carry labels (`env`, `region`, `cloud`, `tenant`, k8s version, detected capabilities). A **ClusterSet** is a label selector over the fleet — the targeting unit for every fleet-wide operation:

- Console: **Fleet → ClusterSets → New**, define the selector, preview member clusters.
- Every tenant should have at least a `canary` set (a small, representative subset) to front every rollout.

## Staged rollouts

Any fleet-wide change runs as a **Rollout**: install/upgrade an operator or Crossplane provider, bump a curated package version, distribute a policy pack, or upgrade agents.

A rollout defines:

| Field | Meaning |
|---|---|
| `target` | What is changing: capability, policy pack, agent version, or catalog version (+ target version) |
| `stages[]` | Ordered stages; each selects a ClusterSet and sets `maxConcurrency` (count or %) |
| `gates` | Optional before/after-stage gates: timed wait or approval (wired into the Approvals module) |
| Health gate | Progression only when agent-reported status is healthy across the stage |

Progression is **health-gated**: the rollout waits for healthy status from the stage's agents before entering the next stage. Stop/resume is supported at any point; **rollback** restores the pre-rollout snapshot version through the same staged machinery.

Recommended first rollout shape: `canary` (1–2 clusters) → approval gate → `wave-1` (≈10%) → timed gate → `remaining`.

## Drift detection (report-only in v1)

Inari continuously diffs desired state (control-plane intent + tenant Git) against reported state (agent capability/status streams). Drift events surface in **Fleet → Drift** and via notifications. v1 is report-only — operators remediate manually or by re-running a rollout; auto-remediation arrives in v1.x.

## Agent upgrade channels

- Desired agent version is set per ClusterSet/channel (`stable`, `canary`); agents auto-upgrade through the GitOps-managed agent manifest on the channel cadence.
- **Compatibility contract:** the control plane supports agent versions **N and N−1** ([ADR-0007](../adr/0007-agent-upgrade-contract-n-minus-1.md)). The console flags out-of-window agents; cordon-and-upgrade them.
- Always roll agent upgrades as a staged rollout — never bump the fleet default channel directly.

## Bulk & ad-hoc operations

Label queries across the fleet ("which clusters run provider-aws v1.x?"), bulk approvals, and bulk policy/catalog assignment are available from the Fleet dashboard.

## Further reading

- [Fleet rollout game-day report](../ops/fleet-rollout-gameday.md) — the M4 exercise of this machinery, including injected failures
- [Cluster lifecycle](cluster-lifecycle.md) — cordon a cluster out of rotation before decommission
- [Policy packs](policy-packs.md) — distributing admission policy via rollout
