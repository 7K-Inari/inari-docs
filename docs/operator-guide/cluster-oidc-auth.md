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

## Verification

`inari-server/e2e/kubectl-access.sh` stands up the full chain (etcd + kube-apiserver + Keycloak, kubelogin exec plugin, viewer-bound group) and asserts: the token carries `groups: ["/tenant-acme/viewers"]`, `kubectl get ns` succeeds, and editor-only operations are denied.
