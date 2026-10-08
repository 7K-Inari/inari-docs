# kubectl access kill switch (runbook)

**Document type:** operational runbook
**Scope:** disabling and re-enabling kubectl access platform-wide or per cluster at runtime ([ADR-0016](../adr/0016-runtime-feature-flags-openfeature.md)).

## Controls

| Level            | Endpoint                                                                                    | Who                                                          |
| ---------------- | ------------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| Platform default | `PUT/DELETE /api/v1/platform/feature-flags/kubectl_access.enabled`                          | platform admin (`org_creator` on `platform:inari`)           |
| Cluster override | `PUT/DELETE /api/v1/tenants/{org}/clusters/{id}/feature-flags/kubectl_access.enabled`       | tenant admin/operator (`clusters.register`)                  |

`PUT` body: `{"value": true|false}`. `DELETE` reverts to the wider scope / built-in default (`true`).

## Precedence

1. `INARI_KUBECTL_ACCESS_ENABLED` **explicitly set** in the environment → env wins; runtime writes are accepted but inert (API responses show `envPinned: true`). The env is **per-binary**: it must be set on BOTH inari-server and inari-kubeproxy, or you recreate the split-brain (proxy 410s while access-info still reports enabled). The Helm value `kubeproxy.kubectlAccessEnabled` (non-null) pins it on both deployments automatically; prefer it over hand-editing one Deployment.
2. Env unset → runtime flag: cluster override → platform default → built-in `true`.

## Effects and latency

- Off: proxy and kubeconfig return **410** ("kubectl access is disabled by platform policy"); new tunnel streams are rejected (`CodeUnavailable`); live tunnel sessions are closed by the kubeproxy `FlagWatcher`.
- Propagation: generation-bump invalidation is immediate with the redis cache backend; with per-process memory caches, convergence within `INARI_CACHE_FLAGS_TTL` (default 10s), plus the watcher poll interval (~2s) for live-session teardown.
- Re-enable: no agent restart needed — the tunnel agent's backoff loop reconnects on its own (worst case bounded by its backoff ceiling).

## Examples

```sh
# Disable kubectl platform-wide
curl -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"value": false}' \
  https://$INARI/api/v1/platform/feature-flags/kubectl_access.enabled

# Disable one cluster only
curl -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"value": false}' \
  https://$INARI/api/v1/tenants/acme/clusters/clu-123/feature-flags/kubectl_access.enabled

# Revert cluster override (inherits the platform value again)
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  https://$INARI/api/v1/tenants/acme/clusters/clu-123/feature-flags/kubectl_access.enabled

# Inspect effective state
curl -H "Authorization: Bearer $TOKEN" https://$INARI/api/v1/platform/feature-flags
```

## Audit

Every write appends `featureflags.set` / `featureflags.cleared` audit rows and a `featureflags.updated` outbox event with actor, scope, and value.

## External flag UI (optional, Flipt)

The chart ships an optional [Flipt](https://www.flipt.io) subchart (Apache-2.0, built-in UI, OIDC login free in OSS, OFREP-native). Enable with `flipt.enabled=true` (or `flagsProvider.url` for a BYO OFREP endpoint) — the server Deployment then gets `INARI_FLAGS_PROVIDER=ofrep` pointing at the subchart service. The Flipt UI manages **external-allowed platform product/rollout flags only**; `kubectl_access.enabled` and any cluster-scoped flag stay DB-authoritative and keep being managed via the API above, never in Flipt. OIDC login for the Flipt UI is configured under `flipt.flipt.config.authentication` (Keycloak example in `values.yaml`).
