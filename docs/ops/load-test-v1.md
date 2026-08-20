# Load test report — v1 scale envelope

**Document type:** M4 report of record
**Scope:** validate the v1 scale envelope (plan §12.1/4, §9 M4): **100 clusters, 5,000 resource instances, 50 concurrent agent streams, agent churn**
**Harness:** load-test scripts live in `inari-server` (`test/load/`); this document is the report of record for methodology, results, and follow-ups.

## Test plan

### Environment

- Control plane: `inari-server` (single replica, then scaled), PostgreSQL, NATS, OpenFGA, Keycloak on the platform cluster (starter-tier-equivalent nodes unless noted).
- Simulated agents: protocol-faithful gRPC clients from `inari-server/test/load`, each holding an `EventStream`, emitting capability/status events at recorded-replay rates and consuming commands with realistic apply latency.
- Dataset: 100 registered clusters; 5,000 `ResourceInstance`s distributed across them; catalog seeded with the v1 curated set + discovered capabilities per cluster.

### Scenarios

| # | Scenario | Load profile | Pass criteria |
|---|---|---|---|
| S1 | Steady state | 100 agents connected; 5k instances reporting status at production-like rates | API p95 within SLO; zero stream drops over 1h |
| S2 | Concurrent stream peak | 50 concurrent agent streams actively syncing (remaining idle-connected) | Agent-gateway CPU/memory within budget; no queue overflow |
| S3 | Agent churn | Rolling disconnect/reconnect storms (10% of agents cycling every 5 min), incl. checksum resync on reconnect | Resync completes per agent; no duplicate/lost commands (at-least-once + idempotency); control-plane latency stays in envelope |
| S4 | Cold start | All 100 agents connect simultaneously after control-plane restart | All streams established; resync completes fleet-wide |
| S5 | API under load | Console/CLI traffic at pilot-fleet levels concurrent with S1 | p95/p99 latency within SLO; OpenFGA check latency consistent with the [M1 spike](../spikes/m1-openfga-performance.md) |

### Metrics collected

API p50/p95/p99 per route class; agent-stream establish rate, drop count, resync duration; per-agent queue depth; command delivery/ack latency; OpenFGA `Check`/`ListObjects` p99 at tuple volume; PostgreSQL/NATS saturation; control-plane CPU/memory per replica.

## Results

:::info Coordination note
Results below are filled from the executed runs (harness: `inari-server/test/load`, run tags referenced per table). Placeholders remain until the final pre-v1.0 run is recorded; earlier iteration results are kept in the run archive linked from `inari-server`.
:::

### S1 — Steady state (run: `<run-tag>`, date: TBD)

| Metric | Target | Result | Pass? |
|---|---|---|---|
| Catalog/resources API p95 | within SLO | TBD | TBD |
| Stream drops over 1h | 0 | TBD | TBD |
| Status event throughput | sustained | TBD | TBD |

### S2 — 50 concurrent streams (run: `<run-tag>`, date: TBD)

| Metric | Target | Result | Pass? |
|---|---|---|---|
| Concurrent active streams | 50 | TBD | TBD |
| Agent-gateway CPU / memory | within budget | TBD | TBD |
| Command ack p95 | within SLO | TBD | TBD |

### S3 — Agent churn (run: `<run-tag>`, date: TBD)

| Metric | Target | Result | Pass? |
|---|---|---|---|
| Resync completion | 100% | TBD | TBD |
| Duplicate commands after reconnect | 0 visible effects (idempotent) | TBD | TBD |
| API p95 during churn | within SLO | TBD | TBD |

### S4 — Cold start (run: `<run-tag>`, date: TBD)

| Metric | Target | Result | Pass? |
|---|---|---|---|
| 100 streams established | all | TBD | TBD |
| Fleet-wide resync duration | recorded | TBD | TBD |

### S5 — API under load (run: `<run-tag>`, date: TBD)

| Metric | Target | Result | Pass? |
|---|---|---|---|
| API p50 / p95 / p99 | within SLO | TBD | TBD |
| OpenFGA `Check` p99 | consistent with [M1 spike](../spikes/m1-openfga-performance.md) | TBD | TBD |

## Findings & follow-ups

_To be completed from the runs. Record: envelope headroom, first bottleneck observed, scaling recommendation (replica counts, per-agent queue sizing), and any file-worthy defects. Cross-reference fixes by commit/PR in `inari-server`._

## Verdict

_Pass/fail against the v1 scale envelope, with conditions. Required for the v1.0 exit gate (plan §9 M4)._
