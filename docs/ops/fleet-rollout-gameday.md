# Fleet rollout game-day — M4 report

**Document type:** M4 game-day record
**Scope:** exercise the staged-rollout machinery end-to-end, including failure injection (plan §9 M4, §5.11; risk: fleet-wide change blast radius, §10).

## Scenario design

Two fleet-wide changes rolled through the staged pipeline against a test fleet with labeled ClusterSets (`canary` → `wave-1` → `remaining`):

1. **Policy pack rollout** — distribute a new `baseline-security` pack version (Kyverno) to the fleet.
2. **Agent upgrade rollout** — bump agents on the `stable` channel to the new version (the riskiest fleet-wide change we ship).

Each rollout used health-gated progression plus one approval gate between canary and wave-1.

## Injected failures

| # | Injection | Expected behavior |
|---|---|---|
| F1 | Canary cluster made unhealthy mid-rollout (agent status forced `Degraded`) | Rollout halts at the health gate; does not enter wave-1; operator notified |
| F2 | Approval gate rejected at wave-1 | Rollout stays paused; rejection audited; resume possible after re-approval |
| F3 | Bad agent version detected in canary (smoke checks failing) | Rollback-to-snapshot restores the previous agent version in canary; fleet untouched beyond canary |
| F4 | Admission webhook from the new policy pack dropped on one cluster | Drift/compliance view flags the cluster; pack rollout reports partial compliance |

## Results

| Check | Result |
|---|---|
| F1 — health gate halted progression | TBD |
| F2 — rejection path audited & resumable | TBD |
| F3 — rollback-by-snapshot restored canary | TBD |
| F4 — compliance gap surfaced per cluster | TBD |
| Happy path: both rollouts completed fleet-wide | TBD |
| Rollback duration (canary) | TBD |

_(Run date: TBD. Fill from the game-day execution; link dashboards/recordings here.)_

## Lessons & follow-ups

_To be completed: gate ergonomics, notification latency, rollback UX, any confusion in per-cluster status reporting. File defects and link PRs._

## Verdict

_Pass/fail on the claim "a bad fleet-wide change cannot reach the whole fleet", with conditions. Feeds the v1.0 exit gate (plan §9 M4) alongside the [load test report](load-test-v1.md) and [upgrade/downgrade drill](upgrade-downgrade-drill.md)._
