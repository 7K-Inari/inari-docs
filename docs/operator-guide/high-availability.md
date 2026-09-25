# High availability

The control plane targets **99.9% availability**. This page is the target posture produced by the HA assessment: the availability budget, the per-component verdicts, the production chart values required to reach them, and the infrastructure SPOFs that must be addressed outside the Inari charts.

:::info Target posture
The HA implementation work (chart knobs, leader-leased loops, golden-path disruption assertions) is landing in parallel across the component repos. Where an item below is still in flight it is described as the **required production posture**, not as shipped default behavior — treat this page as the checklist the charts are converging on, and verify against your chart release before relying on a knob.
:::

## Availability target

| Budget | Value |
| --- | --- |
| Availability target | 99.9% |
| Max downtime per year | ≤ 8.77 h |
| Max downtime per month | ≤ 43.8 min |

**Tolerated disruption:** loss of a single pod or node with **zero failed user requests**. Anything beyond a single replica/node failure (zone loss, dependency outage) is bounded by the [DR runbook](backup-restore.md), not by this budget.

## Component matrix

| Component | HA model | Production posture | Verdict / notes |
| --- | --- | --- | --- |
| `inari-server` | Stateless REST; multi-replica | `replicaCount: 2` + probes, PDB, anti-affinity (chart knobs — in flight) | Replica-safe paths: outbox consumers (`SKIP LOCKED`) for internal/audit delivery; scaffold claim-based dispatch. Coordinated singletons run behind a leader lease (in flight): TZF reconcile, fleet advance/drift, approvals expiry, catalog sync, group syncs, tenant-deletion resume. Migrations are serialized via a PostgreSQL advisory lock; agent streams are fenced server-side, so a reconnect after pod loss cannot split a stream. |
| `inari-operator` | Active-passive via controller leader election (on by default) | `replicaCount: 2` + PDB, anti-affinity | Second replica is a hot standby; leadership failover on pod loss. |
| `inari-console` | Stateless nginx SPA | `replicaCount: 2` + liveness probe, PDB, anti-affinity | No coordination needed; any replica serves any request. |
| `inari-ext-argocd` | Exec mode inherits `inari-server` pod HA | Same as `inari-server` when running in-process | Standalone `http` mode is **not yet deployable** — no chart/manifests exist for it; exec mode is the only HA-supported topology today. |
| `inari-agent` | **Active-passive only** (per tenant cluster) | `replicas: 2` + `leaderElection.enabled=true` | Command dispatch fails closed while the agent is disconnected from the control plane; on reconnect it runs a checksum resync before accepting new commands. A single-replica posture relies on Kubernetes self-healing (restart/reschedule) — no standby, so reconcile pauses until the pod is back. |
| NATS JetStream | 3-node cluster, R=3 streams | 3-node JetStream cluster on the platform cluster | Transient by design — the outbox in PostgreSQL is the source of truth for event replay (see [backup & restore](backup-restore.md)); a full NATS loss is recoverable without backup. |
| Redis (optional) | Shared cache for the authz hot path | Optional; in-memory backend remains the default | `INARI_CACHE_BACKEND=redis` enables a shared PEP/tenant cache across `inari-server` replicas. On Redis outage the server **fails open to direct OpenFGA passthrough** — higher latency on the authz hot path, no availability impact. |

## Required production values

The values below are the production posture for each chart (the HA knobs are being added as part of the HA work — confirm against your chart version). Development installs keep single replicas and may omit these.

### `inari-server`

```yaml
replicaCount: 2
podDisruptionBudget:
  enabled: true
  minAvailable: 1
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - topologyKey: kubernetes.io/hostname
livenessProbe: {}    # chart defaults
readinessProbe: {}   # gate traffic on readiness, not liveness alone
```

### `inari-operator`

```yaml
replicaCount: 2
leaderElection:
  enabled: true      # default; do not disable in production
podDisruptionBudget:
  enabled: true
  minAvailable: 1
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - topologyKey: kubernetes.io/hostname
```

### `inari-console`

```yaml
replicaCount: 2
livenessProbe: {}
podDisruptionBudget:
  enabled: true
  minAvailable: 1
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - topologyKey: kubernetes.io/hostname
```

### `inari-agent` (installed per tenant cluster)

```yaml
replicas: 2
leaderElection:
  enabled: true      # required — agent is active-passive only
```

:::caution Agent replicas
Never run the agent with more than one active replica and leader election disabled. Without the lease, two agents on the same tenant cluster would double-apply commands. The fail-closed dispatch + checksum resync contract assumes a single active agent.
:::

## Infrastructure SPOFs

These are **not** built by the Inari charts — they are the recommended topology for the platform layer the charts deploy onto. Each is a single point of failure for the 99.9% budget if left at its default/dev posture:

| Dependency | Recommended topology | Impact if single-instance |
| --- | --- | --- |
| PostgreSQL | CNPG (CloudNativePG) with ≥ 2 instances, or a managed HA database | Total control-plane outage — all state, outbox, and audit live here |
| Keycloak | HA deployment (≥ 2 replicas, clustered) | No new logins/token issuance; existing sessions degrade |
| OpenFGA | ≥ 2 replicas | Authorization checks fail — every API route enforces OpenFGA |
| Vault | HA mode (Raft or Consul storage); **never dev mode** | Cluster registration exchange fails (`pending_secret_delivery`) |

These roll up into the [production checklist](bootstrap.md#production-checklist) in the bootstrap guide — an HA deployment is not done until every item there and every entry above is satisfied.

## Validation

HA is verified as part of the golden-path test suite: with `INARI_HA=true`, the suite runs disruption assertions — pod kills against `inari-server`, `inari-operator`, and dependencies mid-golden-path — and asserts zero failed user requests and clean leader/stream takeover.

HA test-run evidence (results, disruption timelines, verdicts) is recorded in the [load test report](../ops/load-test-v1.md), alongside the v1 scale-envelope results; the [fleet rollout game-day](../ops/fleet-rollout-gameday.md) shows the drill format used for that class of evidence.
