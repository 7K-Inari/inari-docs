# The catalog

The catalog answers "what can I run here?" — per cluster, from what the cluster **actually has** (plan §5.5). It is machine-maintained by agent capability discovery, so it cannot rot.

## Three sources, one model

Every entry is a `CatalogItem` with an OpenAPI v3 schema, UI hints, a version, and visibility rules:

| Source | What it is | Examples |
|---|---|---|
| **Discovered** | Capabilities the agent found in *this* cluster: CRDs, OLM operator descriptors, Crossplane XRDs/claims, KRO RGDs | `Crossplane provider-aws` you installed yesterday shows up automatically |
| **Curated** | Golden-path packages from `inari-catalog`: KRO `ResourceGraphDefinitions` published as signed OCI artifacts on `stable`/`incubating` channels | "Web service with DNS+TLS", "PostgreSQL on AWS", "namespace-as-a-service" |
| **Platform apps** | Helm/KRO packages and platform-scoped resources (Keycloak realm/client, DNS zone) running on the platform cluster | "Keycloak realm", "cert-manager" |

## Browsing

**Catalog → Browse** shows items filtered to your tenant and the selected cluster's compatibility. Item detail shows documentation, versions, a schema preview, the policy that applies to it, and the **Deploy** action.

The per-cluster view (**Clusters → detail → Capabilities**) is the live, discovered catalog of that cluster — the fastest way to see what a cluster can run right now.

## Versions and pinning

Catalog entries are versioned; your platform team may pin or recommend versions. When a newer version exists for something you deployed, the instance shows a **new version available** badge — see [Resources](resources.md) for the upgrade flow.

## Installing providers

"Install Crossplane provider-aws" is itself a catalog item. Deploying it sends the provider manifests to your cluster via the agent; its new CRDs are then discovered automatically and appear in the catalog. This loop is the signature Inari mechanic: the catalog grows from what you install.

## Visibility & policy

What you see is filtered by tenant visibility rules and your RBAC. Deploy requests are checked by request-time policy — if an item or parameter is blocked, you get the reason and remediation guidance (see [Deploying](deploying.md)).
