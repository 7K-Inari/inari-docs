# Tutorial: Register a cluster

Connect an existing Kubernetes cluster to Inari and watch its discovered capabilities stream into the console — the first "aha". ~15 minutes.

**Prerequisites:** platform-engineer permissions in the console; a Kubernetes cluster with an egress path to the platform's agent-gateway endpoint (outbound 443 is enough); `kubectl` access to that cluster.

## 1. Create the cluster record

Console → **Clusters → Register cluster**:

1. Name it and set labels (`env`, `region`, `tenant`, …) — labels drive ClusterSet targeting later.
2. Submit. The control plane issues a **one-time, TTL'd registration token** and shows an install manifest with the token embedded.

:::warning Token handling
The token is shown once. It is single-use and expires (default TTL is short). If it lapses, re-issue from the cluster detail page — never post it anywhere persistent.
:::

## 2. Install the agent

Apply the manifest to your cluster (or use the Helm chart with the token as a value):

```bash
kubectl apply -f inari-agent-install.yaml
# or
inari cluster register prod-1 --manifest | kubectl apply -f -
```

This installs `inari-agent` with a dedicated tenant-scoped ServiceAccount (read-only capability watches; mutation rights limited to Inari-managed resources).

## 3. Watch the bootstrap

On first connect the agent:

1. Exchanges the registration token for a **per-cluster Keycloak OIDC client** (`cluster-<id>`); the client secret is delivered via ESO, never in Git. The bootstrap token is forgotten.
2. Opens the outbound bidirectional gRPC stream (short-lived JWTs, hardcoded `cluster_id` claim).
3. Starts capability watches: CRDs, OLM CSVs, Crossplane XRDs/providers, Helm releases, KRO RGDs, cluster metadata.

The cluster flips **Pending → Active**. Existing resources are classified **observe-only** by default — nothing is mutated on first connect (brownfield policy, plan §5.3).

## 4. See the catalog build itself

Open **Clusters → your cluster → Capabilities**. Everything the agent found appears as catalog entries within seconds. If the bundle-managed ArgoCD baseline is being installed (default), its CRDs appear as capabilities as they land — the per-cluster catalog growing live is the discovery loop working.

## 5. (Optional) Adopt existing resources

Pre-existing resources stay `observe-only` until you explicitly reclassify them to `adopt` (bring under Inari management) — per resource, audited. Do this from the Capabilities tab after reviewing the ownership implications in the [cluster lifecycle docs](../operator-guide/cluster-lifecycle.md).

## Troubleshooting

| Symptom | Fix |
|---|---|
| Cluster stuck `Pending` | Token expired/used — re-issue and reinstall |
| Agent crash-looping with auth errors | OIDC client delivery failed — check ESO is running in the cluster; re-install pulls a fresh secret |
| Stream connects then drops | Corporate egress proxy interfering with HTTP/2 — see the proxy configuration guidance in the plan §5.3 |
| Cluster shows `Degraded` | Heartbeats missed — check agent logs; it recovers automatically on reconnect |

**Next:** [deploy your first catalog item](first-deploy.md).
