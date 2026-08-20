# Upgrade/downgrade drill — M4 notes

**Document type:** M4 drill record
**Scope:** control-plane upgrade and downgrade, with agent N/N−1 compatibility validation (plan §9 M4, §5.11; [ADR-0007](../adr/0007-agent-upgrade-contract-n-minus-1.md)).

## Procedure under test

1. Start from control-plane version **N** with agents on N and N−1 across a mixed test fleet.
2. **Upgrade** the control plane to N+1 (Helm release upgrade; rolling restart of `inari-server`).
3. Exercise core flows at each step: agent reconnect/resync, catalog browse, deploy, status streaming, approvals, audit writes.
4. **Downgrade** back to N (rollback the Helm release; database migrations must be backward-compatible within the window).
5. Re-exercise the same flows.

## What is validated

- **Rolling upgrade:** no API downtime beyond the rollout; in-flight agent streams reconnect cleanly (backoff + resync).
- **Agent compatibility:** agents at N and N−1 remain fully functional against control plane N+1 — registration, capability events, command dispatch, status stream. Contract CI guarantees the proto surface; the drill confirms runtime behavior.
- **Database migrations:** forward migrations apply online; downgrade works because migrations within the N/N−1 window are backward-compatible (expand-migrate-contract discipline).
- **State integrity:** outbox drains before/after; no lost audit events; OpenFGA tuple reconciler converges after restart.

## Observations (fill per drill)

| Drill date | From → To → From | Upgrade issues | Downgrade issues | Verdict |
|---|---|---|---|---|
| TBD | v0.9.x → v1.0.0 → v0.9.x | TBD | TBD | TBD |

## Follow-ups

_Record any migration that blocked downgrade, streams that failed to resync, or contract gaps found. File defects in `inari-server`; link PRs here._

## Cadence

Run before every minor/major control-plane release. Patch releases run the upgrade leg only. Related: [fleet rollout game-day](fleet-rollout-gameday.md) (agent-side rolling), [DR runbook](../operator-guide/backup-restore.md) (full-restore leg).
