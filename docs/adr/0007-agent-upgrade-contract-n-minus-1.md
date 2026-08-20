# 7. Agent upgrade contract: control plane supports N and N−1

- Status: Accepted
- Date: 2026-08-13
- Deciders: Inari platform engineering
- Source: [Inari platform plan](../architecture/inari-platform-plan.md) §11 Decision 5, applied in §5.11

## Context

Agents auto-upgrade through GitOps-managed manifests on a per-ClusterSet channel cadence (`stable`/`canary`). Control-plane upgrades and agent upgrades cannot be atomic across a fleet, so the compatibility window must be explicit. Supporting an unbounded version skew makes the `inari-api` contract untestable; lockstep upgrades make fleet rollouts fragile.

## Decision

The control plane supports agent versions **N and N−1**. Compatibility is enforced by contract CI against `inari-api`. Agents auto-upgrade on their channel cadence (`stable`, `canary`) via the GitOps-managed agent manifest.

## Consequences

- Fleet rollouts can stage agent upgrades safely (canary ClusterSet first, health gates) — see the [fleet rollout game-day report](../ops/fleet-rollout-gameday.md).
- Contract CI is a hard merge gate in every repo pinning `inari-api`.
- N−2 agents are unsupported: operators must cordon-and-upgrade stragglers; the console surfaces out-of-window agents.
- Validated operationally by the [upgrade/downgrade drill](../ops/upgrade-downgrade-drill.md).
- We would revisit (widening the window) only if real fleets show upgrade cycles longer than one release.
