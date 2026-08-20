# Tutorial: Your first deploy from the catalog

Deploy a curated package to a registered cluster and watch its health stream back. ~20 minutes.

**Prerequisites:** developer access to a tenant with at least one `Active` cluster (see [Register a cluster](register-a-cluster.md)); the `web-service` curated package visible to your tenant on the `stable` channel.

## 1. Find the item

**Catalog → Browse**, filter by your cluster. Choose **Web service with DNS + TLS** — a curated KRO `ResourceGraphDefinition` that composes a Deployment, Service, Ingress, ExternalDNS record, and cert-manager Certificate into one form.

## 2. Fill the wizard

Click **Deploy**:

1. Target cluster: pick your cluster (incompatible ones are filtered out).
2. The form is generated from the package's OpenAPI v3 schema. Set a name, an image, and a hostname. Locked fields (e.g. ingress class, TLS issuer) are set by platform policy — shown read-only.
3. Inline validation enforces allowed ranges; if your platform team set parameter policies, out-of-range values explain themselves.

## 3. Review and submit

The review step shows the rendered manifests. Submit. If the item's approval policy requires it, your request lands in **Approvals** and the rest of this flow continues after approval — you'll be notified (Slack/webhook if configured).

## 4. Follow the GitOps delivery

The orchestrator renders the instance and commits it to your tenant's `<tenant>-inari-state` repository (PR or direct commit per tenant policy), then registers a tenant-local ArgoCD Application via the agent. Nothing is pushed into your cluster from the control plane — your cluster's ArgoCD pulls and applies.

## 5. Watch health in the console

The instance detail page streams status: Application sync/health, KRO instance status, and composed resources going ready one by one (Deployment available → Ingress programmed → DNS record published → certificate issued).

From here you can:

- Deep-link into the tenant-local ArgoCD UI.
- Use the ArgoCD extension's **Sync/Refresh** actions on the instance.
- `kubectl get` the composed resources directly ([kubectl access](../user-guide/kubectl-access.md)) — your tenant-scoped RBAC already covers them.

## 6. Clean up

Delete the instance from its detail page. Teardown is ownership-checked: only resources Inari manages are removed, in reverse-dependency order; the ExternalDNS record and Certificate are cleaned up with the app.

**Next:** [write your first extension](first-extension.md), or explore [upgrades and drift](../user-guide/resources.md).
