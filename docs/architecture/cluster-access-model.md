# Cluster access & authorization model (kubectl)

How a developer's `kubectl` request is authenticated and authorized end to end, and how access is granted and revoked. Code references point into `inari-server` unless noted.

## Identity

- Tenancy is built on **Keycloak Organizations** in one `inari` realm. Users belong to groups `tenant-<slug>/<team>`.
- Kubectl login uses the per-tenant public client **`org-<slug>-kubectl`**, provisioned idempotently at tenant creation by `tenancy.EnsureKubectlClient` (`internal/tenancy/identity.go:204`, called from `tenancy.go`). It allows the standard and device flows, mints tokens with audience **`kubernetes`** and a `groups` claim carrying **full group paths with a leading slash** (`/tenant-<slug>/<team>`) — the exact strings the cluster-side RBAC bindings use as Group subjects.
- Kubeconfigs are secret-free exec-credential configs rendered by `kubeproxy.RenderKubeconfig`, served at `GET /api/v1/tenants/{org}/clusters/{id}/kubeconfig` (or the CLI). The hub never learns the tenant API-server URL (pull-only).

## Authorization model (hub side)

- The permission catalog lives in `internal/authz/permissions.go`. `PermClustersKubectl = "clusters.kubectl"` maps to the OpenFGA relation **`clusters_kubectl` on the organization**. The kubeproxy PEP checks **`RelationKubectl = "kubectl"` on the cluster object** per request (`internal/kubeproxy/proxy.go`), derived from the parent org's `clusters_kubectl` via the FGA model.
- Built-in role bundles (`authz.BuiltinRoles`): **admin and operator include `clusters.kubectl`; editor and viewer do not**. Migration `0031_roles_clusters_kubectl.sql` backfilled existing admin/operator role rows.
- Role and team management go through the tenancy REST API: roles CRUD `/api/v1/tenants/{org}/roles*`, the permission catalog `/api/v1/tenants/{org}/permissions/catalog`, and team→role mappings. FGA tuples are written by the **tuple-writer outbox handler** (`internal/authz/tuplewriter.go`) from role/team/membership events.
- PEP checks are cached by `authz.CachedAuthorizer` keyed `(generation, user, relation, object)` with TTL `INARI_CACHE_PEP_TTL`; every FGA write path goes through `authz.InvalidatingStore`, which bumps the generation. Cache failures fail open to direct OpenFGA.

## Cluster-side enforcement

- The kubeproxy strips any client-supplied `Impersonate-*` headers and **mints its own** from the JWT identity and `/tenant-<slug>/...` group claims.
- The tunnel agent runs with an **impersonate-only** ServiceAccount and forwards the hub-minted headers (`inari-agent`, `internal/tunnelagent/relay.go`); the inbound `Authorization` never reaches the cluster.
- Tenant-cluster RBAC binds the impersonated groups to anchor ClusterRoles `tenant-<slug>-{admin,operator,editor,viewer}` plus per-team role-qualified bindings `tenant-<slug>-<team>-<role>`, materialized by `internal/rbacmaterialize` into `baseline/rbac/` of the tenant's `<slug>-inari-state` repo and synced by the tenant-local ArgoCD.

## Kill switches

Two distinct mechanisms:

1. **`kubectl_access.enabled` runtime flag (kill-switch v2, [ADR-0016](../adr/0016-runtime-feature-flags-openfeature.md))** — reversible, platform- or cluster-scoped, set via the feature-flags REST API. Off → 410 on proxy/kubeconfig and tunnel-stream rejection, plus `FlagWatcher` closing live sessions. An explicitly set `INARI_KUBECTL_ACCESS_ENABLED` env overrides runtime state.
2. **Tunnel-client identity controls** — `tenancy.DisableTunnelClient`/`RevokeTunnelClient` (internal, Keycloak client disable/delete) and the terminal cluster lifecycle endpoints `POST /clusters/{id}/revoke` and `/decommission`.

The UI connect tab reflects the flag via access-info (`kubectlAccessEnabled`, `tunnelAvailable`, `tunnelUnavailableReason`) and shows a read-only "disabled by administrator" notice.
