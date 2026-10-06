# Cluster OIDC authentication (kubectl access)

This is the operator-side contract for [kubectl access via kubelogin](../user-guide/kubectl-access.md) (plan §5.4, §7.2): what a tenant cluster's API server must be configured with so developers can log in with their Keycloak identity. The tenant-zone baseline installs this for you — do not hand-edit it on managed clusters; this page documents the contract so you can audit it or replicate it on brownfield clusters.

## AuthenticationConfiguration

Structured JWT authentication (stable in Kubernetes 1.34) trusting the platform Keycloak `inari` realm, per tenant:

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: AuthenticationConfiguration
jwt:
- issuer:
    url: https://keycloak.example.com/realms/inari
    # PEM of the CA that signs the issuer's TLS certificate.
    certificateAuthority: |
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    audiences:
    - kubernetes            # audience mapper on the kubelogin client
    - org-acme-kubectl      # the tenant's kubelogin client ID (id_token aud)
    audienceMatchPolicy: MatchAny
  claimMappings:
    username:
      claim: preferred_username
      prefix: "keycloak:"
    groups:
      claim: groups
      prefix: ""            # group paths are mapped verbatim
  claimValidationRules:
  # organization is multivalued in Keycloak tokens (["acme"]).
  - expression: 'has(claims.organization) && "acme" in claims.organization'
    message: "token organization does not match this tenant"
```

## Contract points (pinned, plan §5.4)

These strings are the shared contract between token issuance (inari-server), cluster authentication (this file), and RBAC materialization (`rbacmaterialize`). Change them only together:

| Contract | Value |
| --- | --- |
| Groups claim | `groups`, full Keycloak group paths with leading slash, e.g. `/tenant-acme/platform-team` |
| Group mapping | verbatim — no prefix added or stripped by the API server |
| Organization claim | `organization` (multivalued); CEL rule pins it to the tenant alias |
| Audiences | `kubernetes` + `org-<slug>-kubectl` |
| Kubelogin client | `org-<slug>-kubectl` — public, standard + device flow, auto-provisioned per tenant |
| ClusterRoles | `tenant-<slug>-{admin,operator,editor,viewer}` |
| ClusterRoleBinding subjects | `kind: Group`, `name: /tenant-<slug>/<team>` (leading slash included) |

The issuer URL must be **https** — the structured JWT authenticator rejects http issuers. Both the API server and developers' machines (kubelogin) must trust the issuer's TLS CA; kubelogin honors `SSL_CERT_FILE` for custom CAs.

## kubelogin client (auto-provisioned)

inari-server provisions `org-<slug>-kubectl` at tenant creation (idempotent; re-runnable for older tenants):

- public client, standard flow + OAuth2 device authorization grant
- redirect URIs `http://localhost:8000`, `http://localhost:18000` (kubelogin callbacks)
- audience mapper → `kubernetes`; group-membership mapper → claim `groups` with full paths
- `organization` client scope
- no secret — nothing to rotate, nothing on the hub

Developers never configure this by hand: `inari cluster kubeconfig <id>` renders the exec-credential kubeconfig from `GET /api/v1/tenants/{org}/clusters/{id}/access-info`.

## kubectl gateway: tunnel agent identity (plan §7.2)

Gateway mode adds a second identity chain alongside the user-facing one above: a dedicated **tunnel agent** in the tenant cluster dials out to the control-plane kubeproxy and relays tunneled kubectl requests to the local API server.

### Impersonation RBAC

The kubeproxy — never the client — mints `Impersonate-User` (the user's token subject) and `Impersonate-Group` (the token's full group paths, e.g. `/tenant-acme/platform-team`). Client-supplied `Impersonate-*` headers are stripped at the proxy, so the only impersonation identity the API server ever sees on a tunneled request is hub-minted.

The tunnel agent's own ServiceAccount therefore needs **only** the right to pass those headers through:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: inari-tunnel-agent
rules:
- apiGroups: [""]
  resources: ["users", "groups", "uids"]
  verbs: ["impersonate"]
```

The agent holds no API permissions of its own — every tunneled request is authorized as the impersonated user, so the existing tenant ClusterRole bindings apply unchanged. **Never merge the `impersonate` grant into the main inari-agent ServiceAccount**: the fleet agent and the tunnel relay are separate Deployments (same image, `command: ["/inari-tunnel-agent"]` for the tunnel one) with separate SAs, so revoking kubectl tunnel access never touches GitOps delivery.

### Tunnel client provisioning

At cluster registration, inari-server provisions a per-cluster Keycloak client **`tunnel-<cluster-id>`**: confidential, client-credentials grant only, a hardcoded `cluster_id` claim, and an audience mapper pinning `aud=inari-kubeproxy` (`tenancy.EnsureTunnelClient`, idempotent). The kubeproxy stream interceptor rejects any tunnel token without that audience and matching `cluster_id`.

Kill switches, from least to most terminal:

| Action | Effect |
| --- | --- |
| `tenancy.DisableTunnelClient` | Disables the Keycloak client; new tunnel tokens can no longer be minted, in-flight tokens expire on their short TTL. Reversible. |
| `tenancy.RevokeTunnelClient` | Deletes the client outright (cluster revocation path). Terminal. |
| Global flag `kubectl_access.enabled` / `INARI_KUBECTL_ACCESS_ENABLED=false` | Kubeproxy answers 410 to user requests and rejects/closes tunnel streams, platform-wide. |

### Secret delivery (ESO / Vault)

The tunnel client's secret is delivered through the **same mechanism as the agent client secret** (plan §5.3): at registration the control plane writes it to the cluster's Vault path (`secrets.ClusterOIDCPath(cluster.ID)`) under the key **`tunnel-client-secret`** (`agentgateway.TunnelSecretKey`), and the registration response points the agent at an ESO `ExternalSecret` referencing the platform `ClusterSecretStore`. Requirements on the tenant cluster:

- External Secrets Operator installed, with a `ClusterSecretStore` for the platform Vault.
- The rendered `ExternalSecret` must materialize a K8s Secret containing **both** keys: the agent key and `tunnel-client-secret`, in the namespace the tunnel-agent Deployment reads (`INARI_CLIENT_SECRET_FILE`).

If the secret store is not configured on the hub, registration fails with `pending_secret_delivery` and the (unburned) registration token stays consumable — fix the store and let the agent retry.

### Manual bootstrap (no ESO/Vault)

For brownfield clusters where ESO/Vault is not an option, create the Secret by hand after registration:

```bash
# Ask a platform admin for the tunnel-<cluster-id> client secret (rotated fresh via
# tenancy.RotateIdentityClientSecret on tunnel-<cluster-id>), then:
kubectl -n inari-agent create secret generic inari-agent-oidc \
  --from-literal=client-secret=<agent client secret> \
  --from-literal=tunnel-client-secret=<tunnel client secret>
```

Point the tunnel-agent Deployment at it (`INARI_CLIENT_SECRET_FILE=/var/run/secrets/inari/tunnel-client-secret`, mounted from that Secret). Rotation is manual too: rotate the client secret on the hub, update the Secret, restart the tunnel-agent pods. Prefer ESO wherever possible — manual secrets drift and are easy to forget during revocation.

### Upgrading pre-tunnel agents

Clusters registered before the tunnel shipped run an agent without the tunnel relay. Behavior and upgrade path:

1. Access-info reports `tunnelAvailable=false` with a reason; gateway-mode kubectl requests fail fast with **503** and the remediation text "upgrade the inari-agent chart to a version with kubectl tunnel support" — no hang, no partial stream.
2. Upgrade the inari-agent chart in the cluster to a tunnel-capable version. The chart rolls out the additional tunnel-agent Deployment (same image, `command: ["/inari-tunnel-agent"]`, its own SA + the impersonate-only ClusterRole above).
3. The tunnel client `tunnel-<cluster-id>` is provisioned at registration; for clusters registered before this feature, re-run the registration/ensure flow (`tenancy.EnsureTunnelClient` is idempotent) so the client and the `tunnel-client-secret` ESO key are created.
4. Once the tunnel agent connects, its heartbeat row flips `tunnelAvailable` to true and gateway mode starts serving; direct mode worked the whole time and is unaffected.

## Verification

`inari-release-bundle/scripts/e2e/kubectl/kubectl-access.sh` stands up the direct chain (etcd + kube-apiserver + Keycloak, kubelogin exec plugin, viewer-bound group) and asserts: the token carries `groups: ["/tenant-acme/viewers"]`, `kubectl get ns` succeeds, and editor-only operations are denied. `inari-release-bundle/scripts/e2e/kubectl/kubectl-tunnel.sh` covers the gateway chain (kubeproxy + tunnel agent + apiserver): impersonation headers, 503-with-remediation on the upgrade path, the 410 kill switch, per-cluster client revocation, and the max-lifetime reaper.
