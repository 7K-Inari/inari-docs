# Secrets (ExternalSecrets)

Inari uses the **External Secrets Operator (ESO)** with a namespaced `SecretStore` per tenant (plan §4.3). You declare `ExternalSecret` resources; ESO syncs them from your tenant's backend path prefix into Kubernetes Secrets. Nothing secret ever lands in Git or the control plane.

## The model

- Your tenant has a **namespaced `SecretStore`** in each cluster, pre-wired by the tenant-zone baseline to your tenant's backend (e.g. AWS Secrets Manager path prefix `/<tenant>/`).
- The backend policy is scoped to your tenant prefix — you cannot reference another tenant's paths, and admission policies block cross-tenant references.
- A governed `ClusterSecretStore` exists for platform use only; tenants use their namespaced store.

## Declaring a secret

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: app-db-credentials
  namespace: my-app
spec:
  secretStoreRef:
    kind: SecretStore          # namespaced — your tenant store
    name: tenant-secretstore
  target:
    name: app-db-credentials   # the Kubernetes Secret ESO creates
  data:
    - secretKey: password
      remoteRef:
        key: /acme/my-app/db   # inside your tenant path prefix
        property: password
```

Commit this manifest via your application repo or as part of a catalog deployment (several curated packages accept secret references as parameters).

## Verifying

- `kubectl get externalsecret -n my-app` — `READY` and `STATUS` columns show sync state.
- Sync failures (missing backend key, permission denied) surface in the instance's status and as capability-reported admission/ESO status upstream.

## Self-service store onboarding

Bringing your own Vault/AWS SM instance as an additional tenant store is a v1.x feature (plan §7.2). Ask your platform team if you need a non-default backend today.
