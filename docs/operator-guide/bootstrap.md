# Day-0 bootstrap

How the first platform cluster and control plane are installed. Inari never requires Inari to install — bootstrap is plain Helm + a script (plan §12.1/1, M0 exit).

## Prerequisites

- A Kubernetes cluster to become the **platform cluster** (EKS recommended; kind for local/dev).
- `kubectl` context pointed at it, cluster-admin.
- Helm 3, and the OCI registry credentials for `inari/*` images if using a private mirror.
- A DNS zone and TLS issuer you control (for the console + agent gateway endpoints).
- An IdP decision: Inari ships Keycloak as the platform IdP — you need at least one admin identity source (or use Keycloak's local admin for the first login).

## Install

1. **Add the chart repository** (charts ship as OCI artifacts from `inari-helm-charts`):

   ```bash
   helm registry login ghcr.io
   ```

2. **Install the platform baseline chart** — Keycloak (`inari` realm + Organizations), PostgreSQL, NATS, OpenFGA, ESO, cert-manager:

   ```bash
   helm upgrade --install inari-platform oci://ghcr.io/7k-inari/charts/inari-platform \
     --namespace inari-system --create-namespace \
     -f platform-values.yaml
   ```

3. **Install the control plane umbrella chart** — `inari-server` (API, console, agent gateway), `inari-operator`:

   ```bash
   helm upgrade --install inari oci://ghcr.io/7k-inari/charts/inari-control-plane \
     --namespace inari-system \
     -f control-plane-values.yaml
   ```

   Key values: console/agent-gateway hostnames, OIDC issuer URL, initial organization name, image pull policy/signing verification (cosign policy is on by default).

4. **Verify**: console reachable, OIDC login works, `inari-server` health endpoint returns OK, OpenFGA store loaded.

5. **First login & seed**: log in as the platform admin, create the first Organization (tenant), and confirm OpenFGA checks enforce on all API routes.

6. **Run the backup job once and then a restore drill** before onboarding any tenant — see the [DR runbook](backup-restore.md). No tenant onboards before restore is tested.

## Platform Vault setup (cluster registration exchange)

Cluster registration (`agentgateway.RegisterCluster`) delivers the per-cluster OIDC client secret via the platform Vault: the control plane writes `<kvMount>/data/inari/clusters/<cluster-id>/oidc-client-secret` and the tenant cluster's ESO projects it in-cluster. If Vault is not configured, registration fails explicitly with `pending_secret_delivery`.

The control plane authenticates with the **Kubernetes auth method** (ServiceAccount JWT → short-lived Vault token; `inari-server` chart ≥ the release carrying [PR #70](https://github.com/7K-Inari/inari-server/pull/70)). Static token auth remains available for back-compat but expires and needs manual rotation — do not use it for new installs.

### One-time Vault configuration

```bash
export VAULT_ADDR="https://vault.example.org"
vault login

# 1. Write-only policy on the cluster prefix (KV v2)
vault policy write inari-cluster-secrets - <<'EOF'
path "secret/data/inari/clusters/*" {
  capabilities = ["update"]
}
EOF

# 2. Enable Kubernetes auth for the platform cluster (once per cluster)
vault auth enable kubernetes

# 3. Point it at the cluster API (needs a token-reviewer SA with
#    system:auth-delegator, and the cluster CA)
kubectl create serviceaccount vault-token-reviewer -n vault 2>/dev/null || true
kubectl create clusterrolebinding vault-token-reviewer \
  --clusterrole=system:auth-delegator \
  --serviceaccount=vault:vault-token-reviewer 2>/dev/null || true
vault write auth/kubernetes/config \
  kubernetes_host="https://kubernetes.default.svc" \
  kubernetes_ca_cert=@ca.crt \
  token_reviewer_jwt="$(kubectl create token vault-token-reviewer -n vault --duration=8760h)"

# 4. Role for the control plane
vault write auth/kubernetes/role/inari-server \
  bound_service_account_names=inari-server \
  bound_service_account_namespaces=inari \
  policies=inari-cluster-secrets \
  ttl=1h
```

Corresponding chart values (`inari-server`):

```yaml
vault:
  addr: "https://vault.example.org"
  kvMount: secret
  authMethod: kubernetes
  role: inari-server
```

Verify: `vault read auth/kubernetes/role/inari-server`, then register a test cluster and confirm the ExternalSecret in the tenant cluster becomes ready.

### Automating it

Steps 2–3 (auth method + cluster config) are platform-level and belong wherever the Vault instance itself is provisioned (Terraform/OpenTofu Vault provider, or the Vault cluster's own GitOps). Steps 1+4 are Inari-specific and idempotent; if you want them GitOps-managed, run them from a Job with a narrowly-scoped **provisioner token** (never an admin token):

```bash
# provisioner policy: only what the Job needs
vault policy write inari-vault-provisioner - <<'EOF'
path "sys/policies/acl/inari-cluster-secrets" { capabilities = ["create", "update", "read"] }
path "auth/kubernetes/role/inari-server"      { capabilities = ["create", "update", "read"] }
EOF
vault token create -policy=inari-vault-provisioner -ttl=720h -renewable=true
```

The Job mounts that token (ESO-delivered) and runs `vault policy write` + `vault write auth/kubernetes/role/...` — safe to re-run on every sync. Keep the token TTL finite and rotate it like any platform credential.

For development, the same charts install into a kind cluster via the dev-env script in `inari-helm-charts` (`hack/dev-up.sh`). The dev variant disables cosign verification and uses self-signed certs.

## Production checklist

- [ ] External PostgreSQL (or managed) instead of the in-cluster default
- [ ] Backups scheduled + restore drill passed ([DR runbook](backup-restore.md))
- [ ] cosign image-signature verification enabled
- [ ] Agent gateway endpoint on a dedicated hostname with TLS
- [ ] Keycloak admin credentials rotated out of install-time defaults
- [ ] Audit log export target configured
- [ ] Platform Vault configured for the registration exchange (Kubernetes auth role `inari-server`) — see above
- [ ] (Zone vending only) management account connected — see [Tenant Zones](tenant-zones.md)

## Uninstall

`helm uninstall inari inari-platform -n inari-system`. CRDs, PVCs, and the Keycloak/PostgreSQL data survive uninstall by design — delete them explicitly if you intend a clean removal.
