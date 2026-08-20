# Resources

The Resources view is your control plane: every `ResourceInstance` you own, across all your clusters and cloud accounts, in one inventory (plan §5.2 Resources Inventory).

## Inventory

**Deploys / Resources → Resource instances** lists instances with owner team, catalog item + version, cluster, cloud account (if any), and health. Filter by cluster, team, health, or item. If you belong to multiple tenants, the **All tenants** home aggregates this view read-only.

Health is reported by the agent from ArgoCD sync/health and KRO instance status — near-real-time, without the control plane ever polling your cluster.

## Instance detail

Each instance shows:

- Live status and composed resources (with health per resource)
- The catalog item + version it was deployed from, and its rendered spec
- ArgoCD deep link, and extension-provided actions (e.g. sync/rollback from the ArgoCD extension)
- Full event/audit history for the instance

## Upgrades

When the catalog item has a newer version, the instance shows a **new version available** badge. Opening it gives you:

1. A **diff preview** between your rendered spec and the new version's render.
2. One-click upgrade, or hand-off to a [staged fleet rollout](../operator-guide/fleet-rollouts.md) when the same item runs in many clusters.

Upgrades go through the same request-time policy checks as initial deploys.

## Lifecycle

Instances can be deprecated (marked, with a migration notice from the catalog) and torn down. Teardown follows strict ownership semantics — only resources Inari manages are removed, in reverse-dependency order; `observe-only` resources are never touched.

## CLI

```bash
inari resources list --cluster prod-1
inari resources describe <instance-id>
```
