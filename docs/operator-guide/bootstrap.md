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

## Local/dev variant

For development, the same charts install into a kind cluster via the dev-env script in `inari-helm-charts` (`hack/dev-up.sh`). The dev variant disables cosign verification and uses self-signed certs.

## Production checklist

- [ ] External PostgreSQL (or managed) instead of the in-cluster default
- [ ] Backups scheduled + restore drill passed ([DR runbook](backup-restore.md))
- [ ] cosign image-signature verification enabled
- [ ] Agent gateway endpoint on a dedicated hostname with TLS
- [ ] Keycloak admin credentials rotated out of install-time defaults
- [ ] Audit log export target configured
- [ ] (Zone vending only) management account connected — see [Tenant Zones](tenant-zones.md)

## Uninstall

`helm uninstall inari inari-platform -n inari-system`. CRDs, PVCs, and the Keycloak/PostgreSQL data survive uninstall by design — delete them explicitly if you intend a clean removal.
