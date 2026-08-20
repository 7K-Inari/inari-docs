# kubectl access via kubelogin

Tenant clusters trust the platform Keycloak directly via Kubernetes **structured JWT authentication** (`AuthenticationConfiguration`, plan §5.4): issuer is the `inari` realm, audience `kubernetes`, with CEL claim mapping (e.g. requiring the `organization` claim). You use plain `kubectl` with an OIDC login plugin — no kubeconfig files issued by an admin, no copied credentials.

## Setup

1. Install [kubelogin](https://github.com/int128/kubelogin) (`kubectl oidc-login`):

   ```bash
   kubectl krew install oidc-login   # or brew install int128/kubelogin/kubelogin
   ```

2. Get your cluster context from the console: **Clusters → detail → Access** shows the ready-made `kubectl config set-*` commands pre-filled with the issuer URL, client ID, and audience for your tenant. Or use the CLI:

   ```bash
   inari cluster kubeconfig prod-1
   ```

3. First `kubectl` call opens a browser login; kubelogin caches and refreshes tokens afterward.

## How RBAC maps

Your Keycloak **groups** (`tenant-<slug>/<team>`) travel in the token. Each tenant cluster binds those groups to tenant-scoped ClusterRoles — e.g. `tenant-acme-operator`, `tenant-acme-viewer` — via `Group` subjects. The console's RBAC mapping page (cluster detail → RBAC tab) is where platform engineers manage that matrix; membership changes happen in Keycloak only and take effect on your next token refresh.

Control-plane automation acts on your cluster via **impersonation** of tenant-scoped virtual users, so the same RBAC applies uniformly to humans and automation; the audit log records both the real and impersonated identities.

## Troubleshooting

| Symptom | Likely cause / fix |
|---|---|
| `error: You must be logged in to the server (Unauthorized)` | Token expired or missing `organization` claim — re-run `kubectl oidc-login`; confirm you belong to the tenant |
| `Forbidden` on a namespace/verb | Your team is not bound to a ClusterRole allowing it — check with a platform engineer on the RBAC mapping tab |
| Browser never opens | Use `--grant-type=device-code` (kubelogin device flow) on headless machines |
| Works in console, not kubectl | Console permissions (OpenFGA) and cluster RBAC are separate layers — the cluster binding is missing; they are managed from the same groups |

## What not to do

- Do not create long-lived tokens or copy OIDC client secrets into CI. For CI, use a tenant-scoped virtual user via impersonation (ask your platform team) so audit attribution is preserved.
- Do not edit the cluster's `AuthenticationConfiguration` — it is part of the tenant-zone baseline and drift-detected.
