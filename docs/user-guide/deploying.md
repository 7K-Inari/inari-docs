# Deploying from the catalog

Deploying turns a `CatalogItem` into a running `ResourceInstance` in your cluster — rendered from a schema form, policy-checked, delivered by GitOps (plan §5.2 Orchestrator, §5.5).

## The deploy wizard

1. Pick an item and click **Deploy**.
2. Choose the target cluster (only compatible clusters you can access are listed).
3. Fill the form. The wizard walks the item's OpenAPI v3 schema (RJSF), enriched by OLM/KubeVela-style UI hints. Fields your platform team has locked show as read-only with defaults; allowed ranges are validated inline.
4. **Review** — the rendered manifests preview before anything is written anywhere.
5. Submit. Depending on the item's approval policy, the deploy proceeds immediately or waits in the **Approvals** inbox for a peer or platform-admin approval.

## Policy feedback

Every deploy request passes request-time OPA checks. A denial is not a dead end: the response includes the policy that fired, the reason, and what to change — e.g. "image registry `docker.io` is not allowed; use your tenant's ECR registry" (see [policy packs](../operator-guide/policy-packs.md) for how operators write these).

Rendered manifests are checked again at render time (block or warn) before they reach Git.

## GitOps delivery

Rendered instances are committed to your tenant's **`<tenant>-inari-state` repository** ([ADR-0004](../adr/0004-platform-owned-tenant-state-repos.md)) — your application repos are never touched. Whether the render lands as a pull request or a direct commit is a per-tenant policy. Tenant-local ArgoCD (bundle-managed by default) applies the change; nothing is pushed into your cluster from outside.

## Watching it land

After submit, the instance detail page streams status: ArgoCD sync/health, KRO instance status, and composed-resource readiness, reported by the agent in near-real-time. From there you can follow links into ArgoCD, or use kubectl directly (see [kubectl access](kubectl-access.md)).

## CLI equivalent

```bash
inari catalog list --cluster prod-1
inari deploy <catalog-item> --cluster prod-1 --set size=small
inari resources list
```
