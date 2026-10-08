# 16. Runtime feature flags: OpenFeature standard with a DB-authoritative default provider

- Status: Accepted
- Date: 2026-10-08
- Deciders: Inari platform engineering
- Source: kill-switch v2 task (parent drill 3614bd1c, 2026-10-08)

## Context

Kubectl access was gated by `INARI_KUBECTL_ACCESS_ENABLED`, a static env var parsed by `boolEnv` (default true) and consumed independently by inari-server (`clusterregistry` access-info/kubeconfig) and inari-kubeproxy (proxy 410 + tunnel-stream rejection). A live drill proved the failure mode: the flag is **per-binary**, so setting it on kubeproxy alone made the proxy 410 while access-info still reported enabled and the UI notice stayed hidden. There was also no cluster-scoped reversible disable: `tenancy.DisableTunnelClient` existed with zero callers and no REST exposure; the only cluster-scoped controls were terminal (`revoke`, `decommission`).

We needed (a) one runtime source of truth for the kubectl kill switch, (b) platform- and cluster-scoped control settable via API without redeploy, (c) a mechanism future flags adopt instead of new env vars.

Alternatives considered:

1. **DB-backed store only, bespoke read path.** Simple, but invents a proprietary flag API and repeats the mistake of a one-off mechanism per concern.
2. **External flag service as the primary backend (LaunchDarkly/Unleash/flagd).** Rejected as the *default*: a kill switch must work air-gapped with nothing but Postgres; per-cluster scoping is tenancy-adjacent and wants FK integrity plus our audit trail; an external outage must never flip a kill switch.
3. **NATS push invalidation to consumers.** Rejected for v1: inari-kubeproxy deliberately has no NATS wiring, and the shared-cache generation-bump (authz `InvalidatingStore` precedent) already solves cross-replica propagation with TTL-bounded staleness.
4. **OpenFeature (CNCF) SDK as the standard seam, DB provider as built-in default (chosen).** Vendor-neutral: external providers plug in later without touching call sites; the DB provider gives the zero-dependency default.

## Decision

We will make **OpenFeature the standard abstraction for all runtime feature flags** in Inari.

- **Registry.** Every runtime flag is a `Definition` in `internal/featureflags/registry.go` (key, type, allowed scopes, built-in default, description). First flag: `kubectl_access.enabled` (bool, platform+cluster, default true — preserves prior behavior).
- **Storage.** `feature_flags` table (migration 0032): `(flag_key, scope, scope_key)` primary key, `value jsonb`, `updated_by/at`. Platform rows carry `scope_key ''`; cluster rows carry the cluster id.
- **Write path.** `featureflags.Service` persists row + audit + outbox event `featureflags.updated` in one transaction, then bumps the shared generation key `inari:flags:gen` (fail-open; TTL bounds staleness).
- **REST API.** Platform scope: `GET/PUT/DELETE /api/v1/platform/feature-flags[/{key}]` gated on FGA `org_creator` on `platform:inari`. Cluster scope: `GET/PUT/DELETE /api/v1/tenants/{org}/clusters/{id}/feature-flags[/{key}]`, reads on `tenant.read`, writes on `clusters.register` (admin/operator bundles). A dedicated `clusters.configure` permission is deferred.
- **Read path.** All consumers read through an OpenFeature client over the built-in **DB provider** (`featureflags.DBProvider`): evaluation context carries `scope`/`clusterId`; resolution is cluster override → platform row → built-in default. Effective values are cached in the shared cache stamped with the generation (redis: instant cross-replica convergence; memory: TTL `INARI_CACHE_FLAGS_TTL`, default 10s). All store/cache errors **fail open to the built-in default**.
- **Env precedence (hard rule).** `optionalBoolEnv` gives three-state semantics: when `INARI_KUBECTL_ACCESS_ENABLED` is explicitly SET it overrides any runtime value; when unset, runtime state wins. An unparseable value is treated as set to the safe default true. API responses surface `envPinned` so operators can see why a runtime write is inert.
- **Consumers.** inari-kubeproxy evaluates per cluster at the proxy PEP and tunnel admission; a `FlagWatcher` closes live sessions on a flip to off (per cluster via `SessionRegistry.CloseCluster`, platform-wide via `CloseAll`). Re-enable needs no agent restart — the tunnel agent's supervised backoff loop reconnects on its own. `clusterregistry` access-info/kubeconfig report/enforce the per-cluster effective value, eliminating the split-brain.
- **Authority rule.** Kill-switches and cluster-scoped flags are **DB-authoritative even when an external provider is configured**; external providers (a follow-up behind `INARI_FLAGS_PROVIDER`) may serve platform-scoped product/rollout flags only. The data plane never calls an external flag service at runtime.
- **Runtime vs deploy-time boundary.** Helm/chart booleans that shape the deployment (subchart enablement, replica counts) stay chart values; runtime behavior toggles that operators flip during incidents belong in this system. `kubeproxy.kubectlAccessEnabled` becomes nullable: unset renders no env (runtime flag controls); set renders the env (explicit override).

## Consequences

- Kubectl access becomes a single-source, reversible, cluster-scoped control; disabling a cluster denies new kubectl sessions for that cluster only and closes its live tunnel session within the watcher poll interval (~2s) plus cache TTL.
- New runtime flags require a registry entry, not new env plumbing; the REST catalog exposes them automatically.
- The env override escape hatch is preserved for emergencies and air-gapped lockdown.
- Follow-ups (not in this change): external OpenFeature provider wiring; UI admin toggles (the connect-tab notice already reflects the flag via access-info); migration of other flags (UI `features.kubectlAccess` first); a `clusters.configure` permission if cluster flag writes should diverge from cluster lifecycle rights.
- Staleness bound: with the memory cache backend, replicas converge within `INARI_CACHE_FLAGS_TTL`; redis converges on the next read after the generation bump.
