# E2E testing via kubectl proxy

Running end-to-end tests against a tenant cluster from your own machine (or a
CI runner) is easiest through `kubectl proxy`: it exposes the cluster's API on
a local port, authenticated by **your own kubeconfig** — no new credentials,
no VPN, and the control plane never learns your cluster's endpoint
(pull-only). The console walks you through the whole setup.

## Setup in the console

Open **Clusters → &lt;your cluster&gt; → Connect**. The tab guides you through
three steps:

1. **Configure kubectl authentication.** A copyable command adds the
   kubelogin exec credential (issuer URL, your tenant's `org-<slug>-kubectl`
   client ID, audience) to the kubeconfig that already points at your
   cluster's API server. Requires [kubelogin](https://github.com/int128/kubelogin)
   — see [kubectl access via kubelogin](kubectl-access.md) for the underlying
   trust model.
2. **Start the proxy.** A copyable `kubectl proxy --port=8001` command (the
   port is editable). Run it on the machine your tests execute on and leave
   it running.
3. **Verify reachability.** The console polls
   `http://127.0.0.1:<port>/version` directly from your browser and flips to
   **Proxy reachable** (with the cluster's Kubernetes version) once the proxy
   answers. The control plane cannot probe your localhost — this check runs
   entirely in the browser.

Your e2e tests then talk to the cluster through the proxy:

```bash
kubectl proxy --port=8001 &
kubectl --server http://127.0.0.1:8001 get namespaces
# or point your test harness at http://127.0.0.1:8001
```

!!! note "HTTPS consoles"
    If the console itself is served over HTTPS, browsers may block the
    `http://127.0.0.1` probe as mixed content. The indicator then stays in
    the waiting state even though the proxy works — verify with
    `curl http://127.0.0.1:8001/version` in that case.

## Disabling the feature

Two switches compose; the proxy flow is available only when **both** allow it:

- **Global kill switch** — the platform operator sets
  `INARI_DISABLE_KUBECTL_PROXY=true` on the control plane. The Connect tab
  disappears everywhere and `GET /api/v1/features` reports the feature off.
- **Per-cluster setting** — a platform engineer toggles *Disable for this
  cluster* on the Connect tab (or `PATCH /api/v1/tenants/{org}/clusters/{id}`
  with `{"kubectlProxyDisabled": true}`). Other clusters stay usable.

The effective state is computed by the server and returned as
`kubectlProxyEnabled` on cluster payloads — clients never combine the flags
themselves.
