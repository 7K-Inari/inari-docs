# 14. NATS event bus: outbox relay + per-handler JetStream consumer groups

- Status: Accepted
- Date: 2026-09-28
- Deciders: Inari platform engineering
- Source: [Inari platform plan](../architecture/inari-platform-plan.md) §5.2, §5.4 ("outbox → NATS"); supersedes the in-process dispatcher note in inari-server `docs/decisions/0011-db-leader-lease.md`

## Context

Every inari-server mutation writes business rows + audit row + outbox row in one Postgres transaction. Through M0/W1, an **in-process dispatcher** polled the outbox table (`SELECT ... FOR UPDATE SKIP LOCKED`) and invoked registered handlers (OpenFGA tuple writer, rbacmaterialize, resume handlers, notifications) in the same process — the `Publisher` interface was a documented but unimplemented seam for "NATS later". Multi-replica HA worked because every replica ran its own dispatcher over the shared table.

That leaves the plan's event-bus intent (§5.2, §5.4) unrealized at exactly the moment downstream work needs it: the kubectl gateway tunnel wants an ephemeral, low-latency frame bus shared with the inari-kubeproxy service, and external consumers (audit export, notifications) are planned. Alternatives considered:

1. **Dual-mode: keep in-process dispatch as the default, NATS optional (`INARI_NATS_URL` empty = old behavior).** Rejected: two permanent delivery paths with subtly different semantics (dead-letter, ordering, failover) to test and operate forever; the "consumer groups per handler" design would never apply internally.
2. **Additive fan-out: keep in-process handlers, additionally publish to JetStream for external consumers.** Rejected: doubles delivery internally and still builds the durable path only for hypotheticals.
3. **CDC (Debezium/pglogical) instead of an app-level relay.** Rejected: breaks the existing `Publisher` seam, adds infrastructure, and the outbox table + SKIP LOCKED claim loop already solve capture.
4. **Complete transition to NATS JetStream (chosen).** One delivery path, designed for HA from day one — the e2e stack has run a 3-node JetStream cluster since W1 in anticipation.

## Decision

We will make **NATS JetStream the obligatory, single event-delivery path** for inari-server. There is no DB-polled in-process fallback; the in-process dispatcher is deleted.

### Topology and subject namespace

- **Stream `INARI_OUTBOX`**, subjects `inari.outbox.>`, file storage, limits retention (MaxAge 72h — a backpressure bound only; the Postgres outbox table stays the transactional write side and the replay source of truth), `Nats-Msg-Id` dedup keyed by outbox row ID with a 24h window, R = `INARI_NATS_STREAM_REPLICAS` (1 single-node, 3 clustered). Ensured create-or-update at boot so replica changes reconcile online.
- **One durable pull consumer group per handler** (`outbox-<handler-name>`, e.g. `outbox-authz-tuple-writer`): explicit ack, `MaxDeliver` 50 with a capped backoff schedule (1s/2s/5s/10s then 30s), `DeliverAll` so a late-created or renamed durable never misses events. All replicas join the same durables, so JetStream load-balances deliveries cluster-wide — the same exactly-once-ish-per-handler guarantee the HA tests codified for the old dispatcher, now without per-replica polling.
- **Subject convention**: all subjects rooted at `inari.<domain>.<...>`, tokens lowercase without `.`/`*`/`>`/whitespace. `inari.outbox.<event-type>` is durable; `inari.tunnel.<clusterID>.<connID>` is **core-NATS ephemeral** pub/sub (frames ≤ 32 KiB, no persistence, lossy by design) for the kubectl tunnel frame bus. New domains register in the `internal/eventbus` package doc.

### Mechanics

- **Relay**: the dispatcher claims unpublished rows (`FOR UPDATE SKIP LOCKED`, batch 100), publishes each asynchronously **inside the claim TX** with `Nats-Msg-Id = <row id>`, waits for PubAck (10s per-batch budget), and only then marks `published_at`. Crash-after-ack-before-commit republishes and the dedup window collapses it; beyond the window, documented handler idempotency absorbs duplicates. Publishing inside the TX avoids the claim→commit→publish loss window; lock hold time is bounded like the old in-TX handler execution.
- **Fail-open runtime, fail-fast startup** (ADR-0010 philosophy): a NATS outage never crashes the server — relay rows accumulate unpublished and drain on recovery; publish failures increment `attempts`/`last_error` with a 50-attempt dead-letter budget, unchanged. Startup is different: the bus is obligatory, so `Connect` retries for a ~2-minute budget (absorbing first-install ordering with the chart's default-on subchart) and then fails fatally. `/readyz` deliberately never reflects bus state — a bus outage pauses event delivery, not the API.
- **Dead-letter parity**: a terminal handler dead-letter (MaxDeliver exhausted) writes `last_error` back onto the already-published outbox row, plus a `inari.outbox.consumer.deliveries{result="deadletter"}` metric and structured log — one SQL query still finds every dead-letter with its payload co-located for replay.
- **Config**: `INARI_NATS_URL` is required (validation error when empty; comma-separated endpoints accepted); `INARI_NATS_STREAM_REPLICAS` defaults to 1. Metrics at `/metrics`: relay published/errors, unpublished backlog gauge, per-handler deliveries/duration, connection state/reconnects, ephemeral publish errors by domain.
- **Neither relay nor consumers are leader-gated**: SKIP LOCKED claiming and consumer-group load balancing are already multi-replica safe (consistent with ADR-0011's treatment of claim-based loops).
- **Packaging**: the inari-server chart gains the official `nats` subchart, **enabled by default** with single-node JetStream dev posture; production sets `nats.enabled=false` + `nats.url` + `nats.streamReplicas`. The template fails when the subchart is disabled without an external URL. `docker-compose.dev.yaml` gains a `nats` service; gitops runs NATS as a 3-node cluster (`gitops/platform/nats.yaml`) with `streamReplicas: 3` on the inari-server Application.
- **Shared client**: `internal/eventbus` owns the connection, stream provisioning, subject helpers (`OutboxSubject`, `TunnelSubject`, token/frame validation), and ephemeral pub/sub — the seam the kubectl tunnel and inari-kubeproxy will share.

## Consequences

**Easier:**

- HA event delivery with no leader lease and no per-replica polling; pod loss fails deliveries over to surviving replicas via the durable group (validated in e2e HA(a)).
- External consumers can subscribe to `INARI_OUTBOX` with their own durables without touching inari-server code (audit export, notifications fan-out later).
- The kubectl tunnel frame bus and inari-kubeproxy land on an existing, shared, HA substrate with a defined subject convention.
- Multi-replica dev parity: dev, e2e, and production all exercise the same delivery path.

**Harder:**

- NATS is a hard dependency: the server will not boot without it. Mitigated by the default-on subchart, the compose service, the 3-node gitops cluster, startup retry budget, and fail-open runtime semantics.
- Handler execution leaves the claim TX: delivery is at-least-once via JetStream redelivery rather than "handle + mark in one TX". Safe because handler side effects were never in that TX; handlers were already required to be idempotent.
- Rolling upgrade mixes modes transiently: old replicas run the in-process dispatcher, new ones the relay. Safe per row — the SKIP LOCKED claim decides the path exactly once — but dead-letter semantics differ per row during the rollout window. Deploy NATS first (gitops already does; sync waves order it before inari-server).
- The 24h dedup window is finite: a relay crash-loop replaying the same row beyond it can duplicate — covered by handler idempotency.

**Follow-ups (not in scope):** notifications/webhooks as external JetStream consumers; LISTEN/NOTIFY wakeup to cut relay poll latency; stream-based replay tooling (dedicated DLQ stream) if ever wanted; the kubectl tunnel frame bus as the first ephemeral-pub/sub consumer; bumping `gitops/apps/inari-server.yaml` `targetRevision` to the chart release containing NATS support.

**Revisit if:** NATS operational burden (cluster management, PVCs, monitoring) exceeds the operational cost of the old DB-poll status quo, or if event volume outgrows a single limits-retention stream and justifies per-domain streams.

_Note on numbering: inari-server keeps its own `docs/decisions/` series (which contains two documents numbered 0010 — the PEP/tenant cache and the migration advisory lock — plus 0011, the DB leader lease). This ADR continues the inari-docs series; "ADR-0014" below and in code comments refers to this document._
