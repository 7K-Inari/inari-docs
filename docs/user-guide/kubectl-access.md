# kubectl access via kubelogin

Tenant clusters trust the platform Keycloak directly via Kubernetes **structured JWT authentication** (`AuthenticationConfiguration`, plan §5.4): issuer is the `inari` realm, the audiences are `kubernetes` plus your tenant's kubelogin client ID, and a CEL claim-validation rule pins tokens to your tenant's `organization` claim. You use plain `kubectl` with an OIDC login plugin — no kubeconfig files issued by an admin, no copied credentials.

Every tenant gets a kubelogin-ready OIDC client automatically: `org-<slug>-kubectl` (public client, browser + device flow, audience `kubernetes`, groups claim). It appears in the console under **Settings → Identity → Clients** and can be disabled there to revoke kubectl access for the whole tenant.

## Setup

1. Install [kubelogin](https://github.com/int128/kubelogin) (`kubectl oidc-login`):

   ```bash
   kubectl krew install oidc-login   # or brew install int128/kubelogin/kubelogin
   ```

2. Print a kubeconfig with the CLI. Use the cluster ID from `inari cluster list`:

   ```bash
   inari cluster kubeconfig clu-1 --server https://api.prod-1.example.com:6443 > ~/.kube/prod-1
   kubectl --kubeconfig ~/.kube/prod-1 get ns
   ```

   The output contains **no secrets** — only the issuer URL, the public client ID, and exec-plugin arguments. Identity comes from kubelogin at exec time.
   `--server` is required in direct mode: the control plane follows pull-only agents and never learns your cluster's API endpoint — you supply it once.

   On a headless machine (no browser), use the device flow:

   ```bash
   inari cluster kubeconfig clu-1 --server https://api.prod-1.example.com:6443 --grant-type device-code
   ```

3. First `kubectl` call opens a browser login (or prints a device code); kubelogin caches and refreshes tokens afterward.

## Private clusters (pull-only agents)

Direct mode requires your machine to reach the tenant API server. Where the cluster API is private — the normal case, since Inari agents are egress-only — use the **gateway-impersonated** topology (plan §5.4):

```bash
inari cluster kubeconfig clu-1 --gateway > ~/.kube/prod-1
```

kubectl then targets a control-plane proxy endpoint that forwards requests over the agent's outbound gRPC stream with Kubernetes **impersonation headers** (`Impersonate-User` = you, `Impersonate-Group` = your token groups), so the exact same cluster RBAC applies — no second permission model. The proxy itself is a follow-up component; the CLI already renders the correct kubeconfig form for it.

## How RBAC maps

Your Keycloak **groups** travel in the token as full paths with a leading slash (`/tenant-<slug>/<team>`). Each tenant cluster binds those groups to tenant-scoped ClusterRoles — `tenant-acme-admin`, `tenant-acme-operator`, `tenant-acme-editor`, `tenant-acme-viewer` — via `Group` subjects (materialized into the cluster by the RBAC-mappings feature from the same matrix you see under **Settings → RBAC**). Membership changes happen in Keycloak only and take effect on your next token refresh.

Control-plane automation acts on your cluster via **impersonation** of tenant-scoped virtual users, so the same RBAC applies uniformly to humans and automation; the audit log records both the real and impersonated identities.

## Troubleshooting

| Symptom | Likely cause / fix |
|---|---|
| `error: interactiveMode must be specified` | Old kubeconfig rendered before the exec config carried `interactiveMode` — re-run `inari cluster kubeconfig` |
| `error: You must be logged in to the server (Unauthorized)` | Token rejected by the API server: expired (re-run `kubectl oidc-login`), missing `organization` claim (confirm you belong to the tenant), or the cluster's `AuthenticationConfiguration` doesn't trust the `inari` issuer/audiences |
| `oidc: required claim ... / token organization does not match this tenant` | You are not a member of the tenant's Keycloak Organization, or the cluster's CEL rule pins a different tenant alias |
| `Forbidden` on a namespace/verb | Your team is not bound to a ClusterRole allowing it — check with a platform engineer on the RBAC mappings page |
| Browser never opens | Use `--grant-type=device-code` on headless machines |
| Works in console, not kubectl | Console permissions (OpenFGA) and cluster RBAC are separate layers — the cluster binding is missing; they are managed from the same groups |

## What not to do

- Do not create long-lived tokens or copy OIDC client secrets into CI. For CI, use a tenant-scoped virtual user via impersonation (ask your platform team) so audit attribution is preserved.
- Do not edit the cluster's `AuthenticationConfiguration` — it is part of the tenant-zone baseline (see the operator guide) and drift-detected.
