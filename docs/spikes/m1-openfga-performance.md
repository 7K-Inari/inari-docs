# M1 Spike — OpenFGA performance at target scale

**Status:** complete · **Track A (real execution)** · **Recommendation: GO-WITH-CONDITIONS**

Plan references: §5.4 (OpenFGA/ReBAC model), §10 (dual-write drift risk), §12.1/4 (v1 scale envelope: 100 clusters, 5k resource instances), §12.2 item 6.

## Context

Inari puts OpenFGA (Zanzibar ReBAC) on the authorization hot path from v1: the gateway enforces coarse checks, services call `Check` per object and `ListObjects` to filter console/CLI list views. The model is a 6-level inheritance chain — `organization → tenant → cluster → cloud_account → catalog_item → resource_instance` (plus `tenant_zone`) — with tuples referencing `team#member`, never individuals. This spike answers: what are `Check`/`ListObjects` p99 latencies and tuple volume at the v1 scale envelope, and what PEP caching strategy is required? This gates M1 because every later milestone assumes relationship authz on the request path.

## Method

- OpenFGA **v1.18.3** single binary, no authn, HTTP API. Two datastore engines compared: `memory` and `sqlite` (migrated to revision 4). Environment: shared Linux container (30 vCPU, cgroup-throttled), so **absolute numbers are pessimistic**; relative comparisons and the scaling shape are the signal.
- Authorization model: `docs/spikes/harness/openfga/model.json` — mirrors the §5.4 chain, `editor`/`operator`/`viewer` permissions inherited down the chain via `tupleToUserset`.
- Synthetic topology at scale=1 (v1 envelope): 20 orgs × (4 teams + admins), 2 tenants/org, **100 clusters**, 1 cloud account/cluster, 25 catalog items/account, **5,000 resource instances**, ~520 users. Scale=10 = 10× instances/clusters/items.
- Harness (runnable): `cd docs/spikes/harness/openfga && FGA_STORE_ID=$(go run . setup) && go run . load 1 && go run . bench 1 2000 20`. Bench issues randomized `Check(editor|viewer, resource_instance)` and `ListObjects(viewer, resource_instance|cluster)` at fixed concurrency, reporting p50/p95/p99.
- Caching variant: server restarted with `--check-query-cache-enabled --check-query-cache-ttl 10s --check-iterator-cache-enabled`.

## Findings

**Tuple volume.** The envelope produces **13,440 tuples** (~2.7 tuples per resource instance: parent link + owner team); the 10× load produces **127,740 tuples**. Write throughput was unproblematic (~1.1k tuples/s memory, ~1.5k/s sqlite, batched 100/write). Tuple volume is a non-issue at v1 scale — orders of magnitude below any OpenFGA storage limit.

**Latency — measured (shared container, no production tuning):**

| Operation | Datastore | Conc | p50 | p95 | p99 | Notes |
|---|---|---|---|---|---|---|
| Check `editor` on resource_instance | memory | 20 | 124 ms | 302 ms | **423 ms** | envelope, 1× |
| Check `viewer` on resource_instance | memory | 20 | 423 ms | 1,033 ms | **1,201 ms** | union over editor + parent chain |
| ListObjects `viewer` resource_instance | memory | 20 | 239 ms | 454 ms | **569 ms** | 5k objects |
| ListObjects `viewer` cluster | memory | 20 | 35 ms | 87 ms | **101 ms** | 100 objects |
| Check `editor` resource_instance | memory | 20 | 655 ms | 1,251 ms | **1,564 ms** | 10× (127k tuples) |
| Check `viewer` resource_instance | memory | 20 | — | — | >3 s | 10×; server `deadline_exceeded` |
| Check `editor` resource_instance | sqlite | 20 | 892 ms | 1,690 ms | **2,013 ms** | single-writer contention dominates |
| Check `editor` resource_instance | sqlite | **1** | 32 ms | 39 ms | **43 ms** | serial: model cost itself is fine |
| Check `viewer` resource_instance | sqlite | **1** | 133 ms | 153 ms | **164 ms** | serial |
| ListObjects `viewer` resource_instance | sqlite | **1** | 162 ms | 205 ms | **215 ms** | serial |
| Check `editor` resource_instance | sqlite + query cache | 20 | 209 ms | 359 ms | **492 ms** | ~4× p99 improvement vs uncached |
| Check `viewer` resource_instance | sqlite + query cache | 20 | 810 ms | 1,231 ms | **1,416 ms** | random user/object pairs → low hit rate |
| ListObjects `viewer` resource_instance | sqlite + query cache | 20 | 1,542 ms | 2,209 ms | **2,482 ms** | query cache does not cover ListObjects |

**Interpretation.**

1. **The model itself is cheap at envelope scale.** Serial p99: `Check editor` 43 ms, `Check viewer` 164 ms, `ListObjects` 215 ms — on a throttled shared container with a suboptimal datastore. The cost driver is *concurrency × datastore contention*, not graph depth per se. On production hardware with Postgres (the planned deployment, §5.4) these numbers improve substantially; M4's load test on real infrastructure is the confirmation gate.
2. **`viewer` checks are ~4× more expensive than `editor` checks.** The `viewer = this ∪ editor ∪ viewer-from-parent` union multiplies dispatch fan-out across 6 levels. Model design matters: keep deep inheritance on the *write-relevant* relations and flatten read relations where possible (e.g. grant `viewer` directly at the level where it is set, rather than inheriting through the full chain).
3. **`ListObjects` over the largest type is the worst operation and is not helped by the check query cache.** p99 ~0.5–2.5 s at envelope depending on datastore/contention; it degrades fastest at 10×.
4. **Server-side query caching gives a real (~4× p99) win on hot Check paths**, but a randomized access pattern (low locality) still leaves `viewer` p99 > 1 s under load. Real console traffic has high locality (same user, same tenant), so effective hit rates will be far higher than this bench.
5. **The memory datastore must never be used beyond dev**: at 10× it collapses (`deadline_exceeded` on viewer checks). Postgres per the plan; sqlite is not a fallback for anything but demos.

**PEP caching strategy (recommended design).**

- **Gateway/service-level response cache, 1–5 s TTL**, keyed `(user, relation, object)` for `Check`. Staleness bound: the outbox → tuple-writer lag already exists (§10 dual-write risk); a ≤5 s PEP cache adds a bounded, auditable increment. Default-deny on cache errors.
- **Invalidation-on-write where feasible**: the tuple writer emits a `tuple-changed` event on the NATS bus; services with the cache controller pattern invalidate affected keys early. Where infeasible, TTL alone bounds the window.
- **Never serve console list views from `ListObjects` on the request path.** The Resources Inventory module (§5.2) already holds every `ResourceInstance` in PostgreSQL; list views should page from Postgres and filter with **batched `Check`** (MaxChecksPerBatchCheck=50 observed in server config) — 5k-instance worst case becomes 100 batched calls executed concurrently, each ~tens of ms, heavily cache-hit. Reserve `ListObjects` for cold paths (CLI filters, audit tooling).
- Enable OpenFGA **check query cache + iterator cache + cache controller** server-side from day one (measured ~4× p99 on hot paths).

## Recommendation

**GO-WITH-CONDITIONS.** OpenFGA at the v1 scale envelope is viable: tuple volume is trivial, serial latency is healthy, and the hot paths respond well to caching. Conditions:

1. Postgres datastore (never memory/sqlite) + check query cache/iterator cache/cache controller enabled in the platform-cluster OpenFGA deployment; configuration captured in `inari-helm-charts` at M0.
2. PEP cache (1–5 s TTL, invalidation events from the tuple writer) is a **required** component of the `Authorizer` interface implementation, not an optimization.
3. Console/CLI list views must be Postgres-inventory + batched-Check; `ListObjects` is restricted to cold paths. This must be encoded as an `Authorizer` interface guideline before M2 console work.
4. Flatten read-heavy relations in the model where semantics allow (avoid 6-level unions for `viewer`); keep the deep chain for mutation permissions.
5. Re-run this harness against the M4 load-test environment (real Postgres, production-like hardware) and require Check p99 < 100 ms at 10× envelope there.

## Impact if no-go

Not applicable (GO-WITH-CONDITIONS). Had it been no-go, the fallback was already architected: the `Authorizer` interface (§5.4) allows a SpiceDB swap without touching call sites; a harder fallback is precomputed permission tables in PostgreSQL with OpenFGA removed, which would force rewriting §5.4's authz section and deleting the M0 OpenFGA scope — costing roughly a milestone of rework.
