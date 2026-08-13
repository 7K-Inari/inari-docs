# Inari — Internal Developer Platform
## Architecture, Development Plan & V1 Feature Set

**Document type:** Foundational planning / architecture blueprint
**Audience:** Platform engineering leadership, founding engineers
**Status:** v1 draft · 2026-08-13

---

## 1. Executive Summary

**Inari** is a multi-tenant Internal Developer Platform (IDP) built for Development/DevOps teams and Platform Engineers. It is tenant-aware to the core, grounded on a strong IAM/OIDC foundation (Keycloak), and targets public cloud (AWS first) and Kubernetes.

Inari's architecture is a **hub-and-spoke control plane**:

- A **central control plane** (`inari-server`) hosts the API, web console, tenancy/IAM integration, catalog, audit, fleet/policy management, and extension services.
- A lightweight agent (**`inari-agent`**) is installed in each tenant Kubernetes cluster. It connects *outbound only* to the control plane, reports the cluster's **capabilities** (installed operators, CRDs, Crossplane providers), receives desired state, and executes workloads **inside the tenant's own environment** — the control plane never holds tenant cluster credentials.
- A **central platform cluster** runs platform-wide infrastructure (Keycloak, the control plane itself, shared services) and offers **platform apps** (e.g., Keycloak realms/clients, DNS zones) as self-service resources to tenants — and **vends whole Tenant Zones** (new AWS organization account + provisioned tenant cluster + auto-installed baseline) on demand.
- A **capability-driven catalog**: catalog entries are generated from what each cluster *actually has* (discovered by the agent) and from curated packages (KRO `ResourceGraphDefinitions`, Helm charts), so the catalog is machine-maintained and cannot rot.

### Key differentiators (validated against the market)

| Differentiator | Evidence |
|---|---|
| Agent-based **capability discovery** from tenant clusters → auto-generated catalog | No mainstream IDP (Backstage, Port, Cortex, Kratix, KubeVela, Qovery) does this; it directly mitigates "catalog rot," the most reported IDP failure mode |
| **Native multi-tenancy** with per-tenant RBAC, IdP brokering, and tenant-scoped everything | Only Cycloid ("Child Organizations") ships native multi-tenancy; Backstage/Port/Cortex are team-level only |
| **OIDC-first identity fabric** (Keycloak Organizations, per-service audiences, structured k8s auth) | Most IDPs bolt SSO on; Inari makes it the foundation |
| **Tenant Zone vending** — new AWS org account + tenant cluster + baseline in one governed self-service flow | Account vending is normally Control Tower/AFT territory; Inari folds it into the same catalog + policy + lifecycle model |

Native multi-tenancy is near-absent from the portal market — Cycloid's "Child Organizations" is the only native implementation; Backstage, Port and Cortex remain team-level.[^25^] The emphasis on a machine-generated catalog is deliberate: manual catalog upkeep ("catalog rot") is one of the most-reported IDP failure modes.[^26^]
| Everything runs **in the tenant's environment**; control plane is credential-free | Aligns with the argocd-agent/OCM zero-credentials-on-hub security posture |

### Naming note
*Inari* (稲荷) — fitting for a platform that provisions and protects: the kitsune/fox motif is available for branding, and "Inari" is short, pronounceable, and currently unused by major cloud-native projects.

### Open source & business model (decided)
Inari is **fully open source (Apache-2.0)** — no open-core feature gating, every capability ships in the open repos. The commercial model is **services** (consulting, support, managed operations), not product licensing.

---

## 2. Goals → Implementation Alignment

Every stated goal maps to a concrete architectural mechanism:

| # | Goal | Implementation mechanism | Where |
|---|---|---|---|
| G1 | Portal to define platform configuration | `inari-ui` console + `inari-server` Admin API; platform config as versioned CRs on the platform cluster | §5.2, §8 |
| G2 | Curated self-service catalog per tenant cluster | Catalog Service normalizes **discovered capabilities** + **curated packages** (KRO RGDs, Helm) into Catalog Items with OpenAPI-v3 schemas and UI hints; per-tenant visibility rules | §5.5 |
| G3 | Tenant-cluster controller that gathers capabilities | `inari-agent`: watches CRDs, OLM CSVs, Crossplane XRDs/providers; streams `capability-update` events upstream | §5.3 |
| G4 | Register tenant AWS accounts & k8s clusters; all execution stays in the tenant environment | Cluster registration via one-time TTL'd token → per-cluster OIDC client credentials; AWS via `AssumeRoleWithWebIdentity` role created by a one-time CloudFormation/Terraform snippet. Agent executes everything locally via tenant Git + tenant-local ArgoCD | §5.3, §5.7, §6 |
| G5 | Central platform cluster for platform-wide infra (Keycloak) with tenant-consumable resources (realms, clients) | Platform cluster hosts Keycloak + `inari-operator`, which reconciles platform-scoped Catalog Items (Keycloak realm/client, DNS, cert issuers) as tenant resources | §5.6 |
| G6 | Catalog of prebuilt platform applications (Helm, KRO) | `inari-catalog` repo: curated RGD packages + Helm charts published as OCI artifacts, versioned channels | §5.5, §7 |
| G7 | Extensible: backend actions (e.g., ArgoCD actions) and UI extensions | Backend: versioned gRPC plugin contract (go-plugin-style sidecars); UI: Module Federation host + blueprint extension points; plugin backends proxied through an authenticated extension path (ArgoCD proxy-extension model) | §5.8 |
| G8 | Developers install providers (operators, Crossplane providers) into connected clusters | "Provider install" is itself a Catalog Item type; the agent applies provider manifests/GitOps sources into the tenant cluster, then capability discovery picks up the new CRDs automatically | §5.5, §9 |
| G9 | Developers see "what can I run here?" from a generated catalog | Per-cluster catalog view: union of discovered capabilities and curated packages filtered by tenant RBAC and cluster compatibility | §8 |
| G10 | Developer control plane: all resources, connections, integrations | Resources view: `ResourceInstance` inventory streamed back by agents (health from ArgoCD + KRO status), unified across clusters and cloud accounts | §5.2, §8 |
| G11 | Manage lifecycle, policy and the fleet fleet-wide | Fleet Manager + Policy Service modules: ClusterSets & label targeting, staged fleet rollouts (KubeFleet-style, executed credential-free by agents), drift detection, cluster/agent/resource lifecycle states, OPA request-time + render-time checks, Kyverno/CEL policy packs with compliance reporting | §5.11, §7, §9 |
| G12 | Create new Tenant Zones from the Platform zone (new AWS org account + tenant cluster, tenant-zone baseline auto-installed) | Tenant Zone Factory: platform-scoped catalog item (`tenant-zone-aws` KRO RGD) reconciled by Crossplane on the platform cluster (AWS Organizations `CreateAccount` → OIDC-trust bootstrap → EKS) + `inari-operator` wiring registration, Keycloak org, agent/ArgoCD/ESO baseline | §5.12, §7, §9 |

---

## 3. Personas & Consolidated User Stories

### 3.1 Platform Engineer (platform operator/curator)
1. Define development platform configuration in a portal.
2. Curate a self-service catalog; control which items are visible to which tenants/clusters.
3. Run a controller on tenant clusters that gathers capabilities.
4. Onboard tenant cloud accounts (AWS first) and k8s clusters to the control plane, with execution always in the tenant environment.
5. Operate a central platform cluster (Keycloak etc.) and offer platform resources (realms, clients) to tenants.
6. Provide a catalog of prebuilt platform applications (Helm, KRO).
7. Extend the platform with backend actions and UI extensions.
8. Manage the fleet as a fleet: group clusters into sets, roll out operators/providers/policy packs in stages with health and approval gates, and detect drift.
9. Govern with policy as code: request-time guardrails, render-time checks, in-cluster admission packs, and exemptions with expiry.
10. Own lifecycles end-to-end: cluster cordon/decommission, catalog version upgrades, agent upgrade channels, resource teardown with safe ownership semantics.
11. Create new Tenant Zones from the Platform zone: vend a new AWS organization account, provision a tenant Kubernetes cluster in it, and have the tenant-zone baseline (agent, ArgoCD, ESO, policy packs) installed and registered automatically.

### 3.2 Developer (tenant user)
1. Install providers (operators, controllers, Crossplane providers) into connected clusters.
2. Discover what can run on their AWS accounts and clusters from a generated catalog.
3. See all their resources, connections, and integrations in one control plane.

### 3.3 Implicit third persona: Platform Admin / Security
Identity lifecycle, tenant onboarding/offboarding, audit review, extension governance. Explicitly modeled because multi-tenancy makes security workflows first-class.

---

## 4. Architecture Overview

### 4.1 Design principles

1. **Tenant-aware to the core.** Every object carries a tenant ID; every API decision is tenant-scoped; no global namespace collisions.
2. **Zero tenant credentials on the hub.** The control plane stores no tenant cluster kubeconfigs and no cloud keys — only role ARNs, external IDs, and OIDC metadata.
3. **Pull, never push.** Agents dial out; the control plane never initiates connections into tenant networks.
4. **Desired state, eventually reconciled.** All mutations flow as desired state (GitOps or CR-based), not imperative RPCs — agents keep reconciling through network partitions.
5. **The catalog is a projection of reality.** Capabilities are discovered, not declared; curated packages layer on top.
6. **Small kernel, everything else extension.** First-party features (AWS support, ArgoCD actions) are built against the same extension SDK offered to users.
7. **Modular monolith first.** One deployable control-plane binary with strict internal module boundaries; split services only when scaling demands it.

### 4.2 System context (Diagram D1)

![Inari system context and deployment view](diagrams/d1-system-context.png)

Three trust zones:

- **Inari Platform zone** — the platform cluster: `inari-server` (API + console + agent gateway), Keycloak (platform IdP), PostgreSQL, the extension/plugin host, and `inari-operator` managing platform-scoped resources.
- **Tenant zones** — each tenant brings: one or more **k8s clusters** (running `inari-agent`, ArgoCD, optionally Crossplane/ESO/cert-manager) and one or more **AWS accounts** (onboarded via an OIDC web-identity role).
- **Identity zone** — Keycloak `inari` realm with Organizations; optional per-tenant corporate IdPs brokered in.

### 4.3 Technology choices (summary ADR table)

| Area | Decision | Rationale |
|---|---|---|
| Control-plane language | **Go** | Same ecosystem as k8s/client-go; single team competency |
| API style | REST (OpenAPI, code-generated) + gRPC stream for agents | REST for console/CLI; gRPC for the typed, multiplexed agent channel |
| Agent protocol | **Bidirectional gRPC stream**, CloudEvents-style envelope (eventid/resourceid, type, payload) | argocd-agent-proven: typed contract, HTTP/2 keepalive, multiplexing, works outbound-only through restrictive egress |
| Agent auth | One-time TTL'd **registration token** → per-cluster **Keycloak OIDC client** (client-credentials, short-lived JWT, `cluster_id` hardcoded claim) | Unified identity in Keycloak; revocation = disable client; rotation by Keycloak. (mTLS/SPIFFE client certs as a v2 hardening option) |
| IdP | **Keycloak**, one `inari` realm + **Organizations** for tenancy (GA in KC 26; realm-per-tenant collapses operationally past ~hundreds of realms) | Native B2B features: per-org IdP brokering, domain routing, multi-org membership |
| Authorization | **OpenFGA (Zanzibar ReBAC) from v1**, deployed on the platform cluster; gateway=coarse (JWT + org claim), services=fine (`Check`/`ListObjects`); tuples synced from the lifecycle event stream (outbox) | Multi-tenancy + the org→cluster→resource hierarchy make relationship authz core, not optional; OpenFGA chosen for operational simplicity — SpiceDB swappable later if ZedToken-level consistency becomes a hard requirement |
| Packaging/composition format | **KRO `ResourceGraphDefinition`** as the curated-package format | Single YAML, CEL type-checked/terminating (no sandbox needed), generates CRD + controller dynamically, composes any CRDs incl. Crossplane MRs |
| GitOps engine | **ArgoCD** in tenant clusters; agent renders instances to tenant Git + registers Applications | Execution stays tenant-local; status/health stream back via agent informers |
| Cloud provisioning | **Crossplane** (provider-aws), per-account `ProviderConfig`; auth = pod identity via the cluster's OIDC issuer — **IRSA where the cluster is EKS, web identity otherwise** — on whichever cluster runs the provisioning (platform cluster for zone vending/platform resources; tenant cluster for developer-installed providers) | Blast-radius isolation per tenant account; no secrets in etcd |
| Secrets | **External Secrets Operator**: namespaced `SecretStore` per tenant, governed `ClusterSecretStore` for platform | Ecosystem standard; backend policy scoped to tenant path prefix |
| Database | **PostgreSQL** (control plane state), optional NATS for the internal event bus | Single-node start; outbox pattern for events/audit |
| UI | **React + TypeScript + Vite**, Tailwind + shadcn/ui, RJSF for schema-driven forms, **Module Federation** for extensions | Matches extensibility model; schema-driven forms are the catalog's rendering engine |
| Controller framework | **controller-runtime / kubebuilder** for `inari-agent` and `inari-operator` | Ecosystem standard |

---

## 5. Detailed Architecture

### 5.1 Topology at a glance

```
┌─────────────────────────────── INARI PLATFORM ───────────────────────────────┐
│  Platform Cluster (central)                                                  │
│   ├─ inari-server  (API, console host, agent gateway, catalog, audit, authz) │
│   ├─ Keycloak      (realm "inari" + Organizations; platform SSO)             │
│   ├─ PostgreSQL / NATS                                                       │
│   ├─ inari-operator (platform-scoped resources: realms, clients, DNS, ESO)   │
│   └─ Platform apps (Helm/KRO catalog: keycloak, cert-manager, ESO, ArgoCD…)  │
└───────────────▲─────────────────────────────────────────────▲────────────────┘
                │ outbound gRPC stream (agent dials out)      │ OIDC/JWT
   ┌────────────┴─────────┐                      ┌────────────┴─────────┐
   │ Tenant A cluster     │                      │ Tenant B cluster     │
   │  inari-agent, ArgoCD │                      │  inari-agent, ArgoCD │
   │  Crossplane, ESO …   │                      │  KRO, operators …    │
   └──────────┬───────────┘                      └──────────┬───────────┘
              │ AssumeRoleWithWebIdentity                   │
   ┌──────────┴───────────┐                      ┌──────────┴───────────┐
   │ AWS account (A)      │                      │ AWS account (B)      │
   └──────────────────────┘                      └──────────────────────┘
```

### 5.2 Control plane — `inari-server` (modular monolith)

One binary, strict module boundaries, each behind an internal interface so a module can be extracted later.

![Control plane component architecture](diagrams/d2-control-plane.png)

| Module | Responsibility | Key notes |
|---|---|---|
| **API Gateway / BFF** | REST/OpenAPI surface for console & CLI; authn (OIDC JWT validation), coarse authz, rate limits | Gateway = coarse PEP (valid JWT, tenant claim); services = fine PEP |
| **Tenancy & Identity** | Tenant (Keycloak Organization) lifecycle, teams/groups, membership sync, invitations | Tenant ID is a stable `org:<id>`; group paths `tenant-<slug>/<team>` drive k8s RBAC mapping |
| **Cluster Registry** | Registered clusters, registration tokens, per-cluster identity, connection health | Cluster record holds cert/client identity + reported k8s version/labels — never a kubeconfig |
| **Agent Gateway** | gRPC stream termination, per-agent queues, heartbeat, checksum resync, command dispatch | At-least-once delivery + idempotent handlers; agent ID from token claim, never self-asserted |
| **Catalog Service** | Normalizes discovered capabilities + curated packages into Catalog Items; versioning; per-tenant visibility policies; schema/UI-hint storage | Sources: agent capability events, `inari-catalog` OCI artifacts |
| **Orchestrator** | Turns a deploy request into desired state: render RGD instance → tenant Git commit/PR → ArgoCD Application registration (via agent command) | All execution tenant-local; control plane stores intent + status |
| **Cloud Accounts** | AWS account registration (account ID, role ARN, external ID), validation (`sts:AssumeRole` dry-run), per-account Crossplane `ProviderConfig` materialization | Never stores keys |
| **Resources Inventory** | `ResourceInstance` records streamed from agents (owner, catalog item, cluster, health, status) | Feeds the developer control-plane view and future scorecards |
| **Audit** | Immutable audit event log (who/what/where/when) for every action incl. impersonation and agent syncs | Append-only; outbox-written; exportable |
| **Approvals** | Approval workflows on catalog actions (per-item policy: auto / peer / platform-admin) | Governance boundary between tenant autonomy and platform control |
| **Notifications** | Slack/webhook/email on approvals, capability changes, provisioning completion | v1: Slack + generic webhook |
| **Extension Host** | Loads backend plugins (gRPC sidecars, go-plugin-style handshake, versioned contract); exposes plugin endpoints via authenticated reverse-proxy path `/api/extensions/<name>/*` (ArgoCD proxy-extension model) | Plugins inherit Inari authn/authz; first-party extensions (e.g., `inari-ext-argocd`) use the same SDK |
| **Fleet Manager** | ClusterSets (label-based grouping), staged fleet rollouts, agent upgrade channels, drift detection, bulk operations | Health-gated progression from agent-reported status; approval gates via the Approvals module |
| **Tenant Zone Factory** | Vends new tenant zones: AWS Organizations account → OIDC-trust bootstrap → EKS provisioning (Crossplane on the platform cluster) → Inari registration + baseline install | Platform-engineer-only catalog item, approval + policy-gated by default; decommission ties into cluster lifecycle |
| **Policy Service** | One policy model, three enforcement points: request-time (OPA), render-time (orchestrator pipeline), in-cluster admission (Kyverno/CEL policy packs distributed fleet-wide); exemptions with expiry | Unifies catalog visibility/parameter/approval/git policies; compliance readout from agent-reported admission status |

### 5.3 Tenant agent — `inari-agent`

![Agent registration and runtime flow](diagrams/d3-agent-flow.png)

A kubebuilder-built controller deployed by a single install manifest (or Helm) into a tenant cluster.

**Lifecycle**

1. **Register** — Platform engineer creates a cluster in the console → control plane issues a one-time, TTL'd **registration token** (Fleet/OCM pattern)[^1^][^2^]. Install manifest embeds the token. On first connect, the agent exchanges it for a **per-cluster Keycloak OIDC client** (`cluster-<id>`, client-credentials grant); the bootstrap token is forgotten. Client secret delivered to the cluster via ESO — never in git.
2. **Connect** — Agent opens the outbound bidirectional gRPC stream (`EventStream`, the argocd-agent model)[^3^], authenticating with short-lived JWTs (`cluster_id` hardcoded claim). Ping/pong keepalive, backoff reconnect.
3. **Discover** — In-cluster watches on: `apiextensions.k8s.io/v1` CRDs (OpenAPI v3 schemas + CEL validations), OLM `ClusterServiceVersion` descriptors (widget hints), Crossplane XRDs/Compositions/providers, Helm releases, KRO RGDs, plus cluster metadata (k8s version, node labels/taints, installed addon versions). Emits `capability-update` events.
4. **Reconcile desired state** — Receives commands (`apply-bundle`, `register-argocd-app`, `invoke-action`, `render-rgd-instance`); writes rendered manifests to tenant Git (PR or direct commit per tenant policy) and registers tenant-local ArgoCD `Application`s. Imperative out-of-band ops (sync/refresh/rollback, ArgoCD resource actions incl. Lua custom actions) go through the tenant-local ArgoCD API, tunneled over the stream and scoped to Inari-managed resources (fails closed when disconnected).
5. **Report** — Watches Application health/sync + KRO instance status + composed resources; streams `status-update` events upstream. Near-real-time resource inventory without the control plane ever polling tenant APIs.
6. **Survive partitions** — Per-agent queues, idempotent handlers, checksum-based resync on reconnect; the tenant cluster keeps reconciling autonomously while disconnected.

**GitOps decisions (resolved):** (1) The tenant-local GitOps engine is **bundle-managed by default** — the agent installs and lifecycle-manages ArgoCD as part of the tenant-zone baseline — with a **BYO flag** to adopt an existing ArgoCD installation (documented version-skew policy applies). (2) Rendered instances live in a **platform-owned `<tenant>-inari-state` repository per tenant** — application repos stay untouched; PR-vs-direct-commit remains a per-tenant policy. (3) **Git provider:** GitHub first, authenticated via **GitHub App credentials delivered through ESO** (never PATs); GitLab lands in v1.x; the orchestrator hides git behind a provider abstraction from day one.

**Agent RBAC:** dedicated tenant-scoped ServiceAccount; capability watches are read-only; mutation rights limited to namespaces/resources Inari manages.

**Brownfield adoption (decided):** every pre-existing resource/capability found at registration is classified `adopt` (bring under Inari management — explicit, audited, per-resource), `observe-only` (inventory without mutation — **the default**), or `ignore`. Nothing is mutated on first connect.

**Agent footprint budget (decided):** ≤ 100m CPU / 128Mi memory — the agent must fit starter-tier EKS nodes; enforced in agent CI.

**Network constraints (decided):** v1 supports egress-only environments and documents HTTP-proxy configuration for the gRPC stream (HTTP/2 CONNECT); a WebSocket transport fallback and air-gapped operation are deferred to v2 (§12.3).

### 5.4 Multi-tenancy & IAM/OIDC model

![Multi-tenant IAM / OIDC model](diagrams/d4-iam-model.png)

**Tenancy = Keycloak Organization** in one `inari` realm.

- **Organizations** (GA since KC 26)[^5^] give: per-tenant overhead of a DB row (vs a realm object — realm-per-tenant measurably degrades: admin-API p95 ~2ms → 16.6s by 500 realms in published benchmarks[^4^]), per-org IdP brokering with verified-domain routing (enterprise SSO per tenant), and multi-org users (contractors spanning tenants).
- **Tokens**: request the built-in `organization` scope; org-id/attribute mappers put tenant claims in the token. **Services authorize by claim, never by URL/header.** Client scopes shape tokens; one client scope per downstream service with an audience mapper (per-service `aud`) blocks token reuse; Full Scope Allowed off on public clients.
- **Teams = groups** (`tenant-acme/platform-team`) — the vehicle for k8s RBAC mapping, not the tenant boundary.
- **Per-tenant Keycloak realms/clients are self-service catalog resources**, reconciled by `inari-operator` (Keycloak Admin REST / Crossplane provider-keycloak), lifecycle-tied to the tenant. Tenant realms serve *workload* federation (tenant apps' own SSO) — never platform user identity.

**Kubernetes user access (SSO to tenant clusters)**

- Tenant API servers trust the platform Keycloak via **structured JWT authentication** (`AuthenticationConfiguration`, stable in k8s 1.34)[^6^][^7^]: issuer = `…/realms/inari`, audiences `["kubernetes"]`, CEL claim mapping (e.g., require the `organization` claim).
- Users log in with `kubelogin` (`kubectl oidc-login`)[^8^]; tokens carry `groups`.
- **RBAC mapping UX (differentiator):** console maps Keycloak groups → per-tenant ClusterRoles (`tenant-acme-operator`, `tenant-acme-viewer`) bound to `Group` subjects; membership changes live in Keycloak only.
- Control-plane automation acts via **impersonation** of tenant-scoped virtual users, keeping RBAC uniform; audit records both real and impersonated identities.

**Authorization inside Inari — OpenFGA (Zanzibar ReBAC) from v1**[^32^]

- **Model:** `organization → tenant → cluster → cloud_account → catalog_item → resource_instance → tenant_zone`, permissions inherited down the chain (org admin ⇒ tenant admin ⇒ cluster operator ⇒ resource editor); tuples reference teams (`team#member`), never individuals. Org roles (`org-admin`, `platform-engineer`, `developer`, `viewer`) seed the base tuples at tenant creation.
- **Deployment:** OpenFGA runs on the platform cluster — itself installed from the platform-app catalog (dogfooding §5.6).
- **Tuple sync:** the resource-lifecycle event stream (outbox → NATS) drives a tuple writer; a periodic reconciler re-derives tuples from PostgreSQL to heal dual-write drift.
- **PEP split:** gateway = coarse (valid JWT, org claim, route-level); services = fine (`Check` with object id; `ListObjects` to filter console/CLI list views).
- **Why v1, not v2:** tenancy, hierarchy, and cross-tenant guests are day-one requirements for Inari, and retrofitting ReBAC later means rewriting every check. OpenFGA over SpiceDB for operational simplicity; both sit behind the same `Authorizer` interface if ZedToken-grade consistency is ever required.

### 5.5 Catalog & capability discovery

Three catalog sources, one normalized `CatalogItem` model:

1. **Discovered capabilities** — from the agent: CRDs, OLM CSV descriptors, Crossplane XRDs/claims, KRO RGDs present in *this* cluster. Rendered as forms by walking the OpenAPI v3 schema (RJSF-style), enriched by OLM `specDescriptors`/`x-descriptors`[^10^] and optional KubeVela-style UI-schema overrides[^11^] (per-capability hint documents stored as data).
2. **Curated packages** — KRO `ResourceGraphDefinitions` from `inari-catalog` (OCI artifacts, versioned channels `stable`/`incubating`): the golden-path templates ("PostgreSQL on AWS", "Web service with DNS+TLS"). KRO chosen because RGDs are single YAML artifacts, CEL is type-checked/terminating (no template sandbox needed), and kro generates the CRD + controller dynamically on any tenant cluster.[^9^] Raw discovered capabilities can be *wrapped* into curated RGDs over time.
3. **Platform apps** — Helm charts/KRO packages installable on the **platform cluster** (Keycloak itself, cert-manager, ESO, ArgoCD, monitoring stack), plus platform-scoped resource types (Keycloak realm, Keycloak client) reconciled by `inari-operator`.

**Curation model:** platform engineers assign visibility (which tenants/clusters see which items), parameter policies (locked defaults, allowed ranges), approval policy, and GitOps target policy. Catalog entries are versioned; tenants pin versions.

**Provider installs (developer story D1):** "install Crossplane provider-aws" is itself a Catalog Item whose payload is the provider manifest/GitOps source. Agent installs it → its CRDs appear → capability discovery emits them → the catalog grows automatically. This loop is Inari's signature mechanic.

### 5.6 Platform cluster & `inari-operator`

The platform cluster is both the **host of the control plane** and a **first-class deployment target**:

- Runs Keycloak, PostgreSQL/NATS, the control plane, and platform apps from the curated catalog (Helm/KRO).
- `inari-operator` reconciles **platform-scoped Catalog Items** requested by tenants: Keycloak realm/client, DNS zone/records (ExternalDNS), cert issuers, shared ArgoCD projects, tenant namespaces on the platform cluster (for tenants allowed to run workloads centrally).
- Tenant access to platform-cluster resources is namespace-isolated and mediated by impersonation (§5.4) — the same authz model as tenant clusters.

### 5.7 AWS account onboarding

1. Tenant admin clicks "Connect AWS account" → console shows a one-time **CloudFormation/Terraform snippet** creating an IAM role that trusts the platform cluster's OIDC provider (`sts:AssumeRoleWithWebIdentity`, conditioned on `sub` = the Crossplane/agent service account, `aud = sts.amazonaws.com`, optional `ExternalId`).
2. Control plane stores **only** account ID, role ARN, external ID, issuer metadata. Validation = a dry-run assume-role.
3. Crossplane per-account **`ProviderConfig`** pinned via `providerConfigRef`; `source: IRSA` platform-side.[^12^] Tenant A's managed resources can only act in tenant A's account.
4. Bootstrap role is **least-privilege** (scoped to services Inari manages).[^13^] Tenant revokes by deleting the role.

**Where Crossplane runs, and what "platform side" means.** There are two run contexts, both keyless:
- **Platform cluster (Inari-operated)** — used by the Tenant Zone Factory (§5.12) and platform-scoped cloud resources. `ProviderConfig.spec.credentials.source: IRSA`: the provider pod's ServiceAccount is annotated with the platform bootstrap role ARN; the platform cluster's EKS OIDC issuer vends it a web-identity token, and it assumes the role via `sts:AssumeRoleWithWebIdentity`. To act **inside a tenant account**, the per-account ProviderConfig then role-chains into that account's onboarding role (which trusts the platform cluster's OIDC issuer, conditioned on `sub`/`aud`). Nothing long-lived is stored on either side.
- **Tenant cluster (developer-installed from the catalog)** — manages the tenant's own AWS resources in their own account. If the tenant cluster is EKS in that account, this is plain IRSA against the *tenant cluster's* OIDC issuer; if the cluster is not EKS, the same web-identity flow applies, provided the cluster's OIDC issuer is publicly reachable (e.g., S3-hosted discovery document).

IRSA is simply EKS's packaging of web-identity federation; "platform side" vs "tenant side" only changes **whose OIDC issuer the AWS role trusts**.
5. Account vending ships in v1 via the Tenant Zone Factory (§5.12) — bring-your-own and vended accounts converge on the same OIDC role-trust contract. Control Tower/AFT coexistence (importing AFT-vended accounts) is v2.

Azure/GCP later via the same `CloudAccount` abstraction (different trust mechanics, identical model).

### 5.8 Extensibility model

**Backend extensions** — versioned gRPC contract (`inari-plugin-sdk`): plugins run as sidecar containers/subprocesses (hashicorp go-plugin model: handshake cookie, protocol version, checksum verification, crash isolation)[^14^]. Use cases: ArgoCD actions, custom capability detectors, external inventory importers. Plugin HTTP endpoints surface through `/api/extensions/<name>/*` — the control plane authenticates, enforces RBAC (`extensions, invoke, <name>`), strips sensitive headers, reverse-proxies (ArgoCD proxy-extension pattern)[^15^]. WASM reserved for future untrusted/tenant-supplied extensions.

**UI extensions** — Module Federation host[^16^] + manifest-driven remote registry (`remoteEntry.js` served via backend/OCI). A **blueprint/extension-point contract** (Backstage new-frontend style) instead of free-form rendering: typed slots — `NavItemBlueprint`, `CatalogCardBlueprint`, `ClusterTabBlueprint`, `FormWidgetBlueprint`, `PageBlueprint`. Shared React singletons; remotes registered at runtime.

**First-party eats the same dogfood:** the ArgoCD action extension and AWS provider support are built as extensions on the public SDK.

### 5.9 Data model (core entities)

```
Organization (tenant) 1───n Team ─── n User(refs Keycloak)
Organization 1───n Cluster ─── n Capability (discovered, versioned, managementMode: adopt|observe|ignore)
Organization 1───n CloudAccount (aws: accountId, roleArn, externalId)
CatalogItem (source: discovered|curated|platform; schema; uiHints; version; visibility[])
ResourceInstance (catalogItemId, clusterId, cloudAccountId?, spec, health, status, ownerTeam, managementMode)
Environment (v1.x: logical grouping + TTL)
ApprovalRequest (action, requester, approver, state)
AuditEvent (actor, impersonator?, action, object, tenant, ts)
RegistrationToken (clusterId, ttl, one-time)
Extension (name, version, kind: backend|ui, manifest)
PlatformApp (catalogItemId targeting platform cluster)
ClusterSet (name, labelSelector, memberClusterIds)
Rollout (target: capability|policy-pack|agent|catalog-version, stages[], gates, status, snapshotRef)
PolicyPack (ociRef, version, engine: kyverno|cel-vap, parameters)
PolicyAssignment (policyPackId → ClusterSet|tenant, exemptions[])
Exemption (policyId, scope, reason, expiresAt, approvedBy)
DriftEvent (clusterId, objectRef, desiredHash, reportedHash, detectedAt)
TenantZone (zoneId → orgId, cloudAccountId, clusterId, tier, state, provisionedBy)
```

### 5.10 Security model summary

- No tenant credentials at rest on the hub (no kubeconfigs, no cloud keys).
- All agent identity from short-lived OIDC JWTs with hardcoded `cluster_id`; bootstrap tokens are one-time + TTL'd.
- All cross-boundary cloud auth via short-lived assumed-role sessions.
- Per-tenant isolation: Keycloak Organizations, per-cluster control-plane partitioning, per-account ProviderConfigs, namespaced ESO SecretStores, admission policies blocking cross-tenant references.
- Every mutation (incl. impersonated) emits an immutable audit event.
- Supply chain: OCI-signed catalog artifacts (cosign), checksum-verified plugin binaries, SLSA provenance on release images.

### 5.11 Fleet, Lifecycle & Policy Management

**Fleet management — the Fleet Manager module.** Registered clusters carry labels (`env`, `region`, `cloud`, `tenant`, k8s version, detected capabilities); label selectors define **ClusterSets** — the targeting unit for every fleet-wide operation.

- **Staged fleet rollouts.** Any fleet-wide change — install/upgrade an operator or Crossplane provider, bump a curated package version, distribute a policy pack, upgrade agents — runs as a staged rollout: stages select ClusterSets by label, each stage sets `maxConcurrency` (count or %) and optional before/after-stage gates (timed wait or approval, wired into the Approvals module), with **health-gated progression** driven by agent-reported status; stop/resume and rollback-to-previous-version supported. The model follows KubeFleet's staged update runs[^27^][^28^], but executes **credential-free**: each stage hands desired state to the target clusters' agents, which apply it via tenant-local GitOps. (OCM/KubeFleet place work into per-cluster hub namespaces that agents pull[^2^]; Inari's agent stream carries that role without any hub-side cluster state.)
- **Agent lifecycle (upgrade contract, decided).** Desired agent version per ClusterSet/channel (`stable`, `canary`); the **control plane supports agent versions N and N−1**, enforced by contract CI against `inari-api`; agents auto-upgrade through the GitOps-managed agent manifest on their channel cadence.
- **Drift detection.** Continuous diff of desired (control-plane intent + tenant Git) vs reported (agent capability/status streams); drift events surface in console and notifications. v1 is report-only; auto-remediation is v1.x.
- **Bulk & ad-hoc ops.** Label queries across the fleet ("which clusters run provider-aws v1.x?"), bulk approvals, bulk policy/catalog assignment.

**Lifecycle management.**

- **Clusters:** `Pending → Active → Degraded → Cordoned → Decommissioned`. Cordon blocks new deploys while existing workloads keep running. Decommission drains Inari-managed resources in reverse-dependency order (ownership-checked), revokes the cluster's OIDC client, and archives the audit trail.
- **Resource instances:** provision → update (catalog "new version available" badge, diff preview, one-click or staged upgrade) → deprecate → teardown with strict ownership semantics (no shared-tenant resources by default — §10).
- **Catalog & platform apps:** versioned channels with deprecations and migration notices; platform-cluster apps upgrade through the same staged-rollout machinery as tenant clusters.

**Policy management — the Policy Service module.** One policy model, three enforcement points:

1. **Request-time (pre-flight):** OPA evaluation of every catalog deploy/update request against tenant + cluster policies — allowed registries, required labels, size ceilings, cost guardrails, approval requirements[^29^]. Developers see the *reason* and remediation guidance, not just a denial.
2. **Render-time:** policy checks on rendered manifests in the orchestrator pipeline (block or warn) before anything reaches tenant Git.
3. **In-cluster admission:** versioned **policy packs** (Kyverno policies or CEL ValidatingAdmissionPolicies) distributed to ClusterSets via fleet rollout, sourced from `inari-catalog` (e.g. `baseline-security`, `cost-guardrails`)[^30^][^31^]. Because packs and admission webhooks are agent-discovered capabilities, compliance is observable: the agent reports admission status upstream, feeding a per-cluster/per-tenant compliance view.

**Exemptions** are time-boxed, approval-gated, and fully audited. The catalog visibility/parameter/approval/git policies from §5.5 are unified under this service — one policy surface, not four.

**V1 scope:** ClusterSets & label targeting; staged rollouts for capabilities/policy packs/agents; drift report-only; cluster lifecycle states; resource upgrade flow; request-time OPA + policy pack distribution. Deferred: drift auto-remediation, cost-based policies, multi-language policy authoring.

### 5.12 Tenant Zone Factory — creating new Tenant Zones from the Platform zone

A platform-engineer-only, platform-scoped Catalog Item (`tenant-zone-aws`, packaged as a KRO RGD in `inari-catalog`) that turns "new tenant" into one governed, auditable request. Execution happens on the platform cluster; the result is a fully registered, baseline-installed Tenant Zone.

**Prerequisite:** the platform cluster is connected to the AWS Organizations **management account** — a special `CloudAccount` record (`scope: management`) whose role allows `organizations:CreateAccount/TagResource/DescribeCreateAccountStatus` (and account-closure actions for decommission). Least-privilege: nothing else.

**Provisioning flow:**

1. **Request** — PE fills the zone form (name/slug, OU placement, region, cluster tier); the Policy Service pre-flights (naming rules, OU quota, budget guardrails); default policy: approval required.
2. **Account vend** — Crossplane (`provider-aws-organizations`) on the platform cluster creates the AWS account in the target OU and waits for readiness (async; minutes)[^33^][^34^]. A baseline policy pack applies org-side guardrails (CloudTrail, budget alert, mandatory cost tags).
3. **Trust bootstrap** — via the auto-created `OrganizationAccountAccessRole`, Inari creates the standard OIDC web-identity role in the new account (the same contract as BYO onboarding, §5.7). From here on everything uses `AssumeRoleWithWebIdentity` — no stored keys.
4. **Cluster provision** — Crossplane EKS resources (cluster, node group, OIDC provider for IRSA) materialize the tenant cluster per the requested tier.
5. **Inari wiring** (`inari-operator` + control plane): create the Keycloak Organization + default groups and RBAC mapping; create Cluster + CloudAccount records; issue a registration token; render the tenant-zone baseline bundle (inari-agent, ArgoCD, ESO, baseline policy packs) into the zone's new Git repo and register the tenant-local ArgoCD root app. The agent connects, exchanges the token, and the bootstrap credential is forgotten.
6. **Active** — the zone flips to `Active`; capabilities start streaming; the curated catalog becomes visible per tenant policy.

**Lifecycle ties:** decommissioning a Tenant Zone reverses the flow — cordon → drain Inari-managed resources → delete EKS → `CloseAccount`/suspend → revoke identities → archive audit — behind approval gates with ownership checks (§5.11).

**Guardrails:** org/OU quota pre-checks, region allow-lists, mandatory cost tags enforced at request time; every step emits audit events; the flow is resumable and idempotent (long-running AWS operations tracked as sub-resources with status).

**v1 scope:** AWS-only, EKS-only, new accounts under an existing AWS Organization. Ships a single **starter tier** (one region; EKS with a minimal managed node group, e.g. 2× t3.large) and requests only the **minimum AWS account-creation quota** needed to start — additional tiers land with demand. Later: import existing org accounts, Control Tower/AFT coexistence, additional cluster distros, Azure/GCP zones.

---

## 6. Repository Topology

Polyrepo under a single GitHub org, e.g. **`inari-dev`** (mirrors how k8s ecosystem projects scale: one repo per independently-versioned artifact). Naming convention: `inari-<part>`; the org prefix keeps repo names short.

![Repository topology map](diagrams/d6-repo-map.png)

| # | Repo | Contents | Stack | Versioned artifact |
|---|---|---|---|---|
| 1 | **`inari-server`** | Control plane: REST API/BFF, agent gRPC gateway, tenancy, cluster registry, catalog service, orchestrator, cloud accounts, resources inventory, audit, approvals, notifications, extension host | Go, PostgreSQL, chi/Huma, NATS | Container image `inari/server` |
| 2 | **`inari-agent`** | Tenant-cluster controller: registration/bootstrap, capability discovery watches, gRPC client, GitOps renderer, ArgoCD command proxy, status streamer | Go, controller-runtime | Container image `inari/agent` + install manifests |
| 3 | **`inari-operator`** | Platform-cluster operator: tenant Keycloak realms/clients, platform namespaces, DNS/cert shared resources, platform app installs | Go, controller-runtime | Container image `inari/operator` |
| 4 | **`inari-ui`** | Web console (host shell, all first-party pages), design system, schema-form renderer | React, TS, Vite, Tailwind, shadcn/ui, RJSF, Module Federation | Static bundle served by `inari-server` |
| 5 | **`inari-api`** | Protobuf (agent protocol, plugin contract) + OpenAPI specs; generated Go/TS clients | Buf, protoc, oapi-codegen | Versioned contract packages |
| 6 | **`inari-plugin-sdk`** | Go SDK for backend extensions (handshake, lifecycle, auth context, helpers) | Go, hashicorp go-plugin | Go module |
| 7 | **`inari-ui-plugin-sdk`** | TS SDK: extension-point blueprints, host APIs, dev harness (run an extension standalone against a dev control plane) | TS | npm package |
| 8 | **`inari-catalog`** | Curated packages: KRO RGDs, platform-app Helm charts, UI-schema hints, per-package docs/tests; CI publishes signed OCI artifacts | YAML, CEL, Helm, OPA tests | OCI artifacts + channels (`stable`, `incubating`) |
| 9 | **`inari-cli`** | `inari` CLI: login (OIDC device flow), cluster/catalog/resource ops, agent install, extension scaffolding (`inari extension init`) | Go, cobra | Binary releases (brew/scoop/go install) |
| 10 | **`inari-helm-charts`** | Deployment: control-plane umbrella chart, agent chart, platform-cluster baseline chart | Helm | Chart releases (OCI) |
| 11 | **`inari-ext-argocd`** | Reference + first-party extension: ArgoCD actions (sync/refresh/rollback/custom Lua actions), ArgoCD status cards/tabs for the UI | Go + TS | Container image + UI remote |
| 12 | **`inari-docs`** | Docs site, ADRs, user/operator/extension-author guides, tutorials | Astro/Docusaurus | Static site |

**Repo-count rationale:** 12 repos total, but only **4 runtime components** (server, agent, operator, UI) and **2 SDKs**; the rest are contracts (api), content (catalog, charts, docs), tooling (cli), and the reference extension. If the team is < 4 engineers at start, merge 5→1 (api into server) and 7→4 (UI SDK into UI) temporarily, keeping directory boundaries — reducing to **10** with zero structural change later.

**Cross-repo mechanics:** `inari-api` is the compatibility contract — server, agent, cli, and plugins pin its versions; CI runs contract tests against it. `inari-catalog` has its own release cadence (content ships faster than binaries). Every repo: conventional commits, semantic versioning, cosign-signed images/artifacts.

---

## 7. V1 Feature Set

MoSCoW prioritization. **M = v1.0 must**, S = v1.0 should (first to slip), C = v1.x, W = explicitly deferred.

### 7.1 Platform Engineer features

| Feature | Priority |
|---|---|
| Platform configuration portal (org settings, tenancy, IdP, policies) | M |
| Cluster registration workflow (token issuance, install manifest/Helm, connection health) | M |
| Agent-based capability discovery (CRDs, OLM descriptors, XRDs, KRO RGDs, versions) | M |
| Catalog curation: visibility per tenant/cluster, parameter policies, approval policy, version pinning | M |
| Curated catalog v1: ~8–12 golden-path packages (KRO RGDs) — e.g., web-service+DNS+TLS, PostgreSQL (AWS), S3-backed app, namespace-as-a-service, Keycloak realm/client | M |
| Platform cluster + platform apps catalog (Keycloak, cert-manager, ESO, ArgoCD, monitoring) | M |
| DNS/ingress/TLS automation as tenant-facing catalog capabilities (ExternalDNS + cert-manager + ingress controller) | S |
| Platform-scoped tenant resources: Keycloak realms/clients, DNS, tenant namespaces (via `inari-operator`) | M |
| AWS account onboarding (OIDC web-identity role, validation, per-account ProviderConfig) | M |
| Audit log (every action, filterable, exportable) | M |
| Approval workflows on catalog actions | M |
| RBAC mapping UX (Keycloak groups → tenant cluster roles) | M |
| Backend extension host + `/api/extensions/*` proxy; UI extension slots (nav, catalog cards, cluster tabs) | M (host) / S (polish) |
| Provider/operator install as catalog items (into tenant clusters) | M |
| Fleet management: ClusterSets (label targeting), fleet dashboard | M |
| Staged fleet rollouts (canary → waves, health gates, approval gates, rollback) for capabilities, policy packs, agent upgrades | M |
| Agent upgrade channels (stable/canary) + N/N−1 compatibility | M |
| Drift detection (desired vs reported; report-only in v1) | M |
| Cluster lifecycle: states, cordon, decommission (drain + identity revocation + archived audit) | M |
| Resource instance upgrades (version badge, diff preview, staged upgrade) | M |
| Policy service: request-time OPA checks + render-time manifest checks | M |
| Policy packs (Kyverno / CEL ValidatingAdmissionPolicies) distributed to ClusterSets; compliance view | M (distribution) / S (compliance) |
| Policy exemptions (time-boxed, approval-gated, audited) | S |
| **Tenant Zone Factory**: vend new AWS org account + EKS tenant cluster + auto-installed baseline (agent, ArgoCD, ESO, policy packs), registered end-to-end | M |
| OpenFGA relationship-based authorization (org→tenant→cluster→catalog→resource), tuples synced via outbox | M |
| Drift auto-remediation | C (v1.x) |
| Cost-aware/budget policies | C (v1.2) |
| Notifications (Slack + webhook: approvals, capability changes, provisioning done) | S |
| Scaffolding/templates (repo + pipeline + catalog entry + tenant RBAC in one flow) | S |
| Scorecards fed by capability data | S (v1.1) |
| Environment lifecycle (ephemeral envs, TTL auto-teardown) | S (v1.1–v1.2) |
| Cost visibility per tenant (AWS CUR/OpenCost) | S (v1.2, differentiator) |
| Adoption/usage dashboards | S (v1.2) |
| Observability deep-links (Grafana/Prometheus links per resource; no built-in dashboards) | S (v1.2) |
| Multi-cloud beyond AWS; service mesh; high-compliance modes; account vending | W (v2+) |

### 7.2 Developer features

| Feature | Priority |
|---|---|
| SSO login via platform OIDC; tenant/team scoping everywhere | M |
| Per-cluster "what can I run here?" catalog (discovered + curated, RBAC-filtered) | M |
| Schema-driven deploy wizard (RJSF forms from CRD/RGD schemas, validation, locked fields) | M |
| Request-time policy feedback (why a deploy was blocked, what to change) | M |
| Install providers/operators/Crossplane providers into my clusters from the catalog | M |
| Resources control plane: all my instances across clusters/accounts with health & status | M |
| GitOps-native deploys (PR or direct commit per tenant policy); status streams back | M |
| kubectl access via kubelogin with tenant-scoped groups | M |
| Secrets: declare ExternalSecrets against tenant SecretStore | M |
| `inari` CLI — core flows only: login, cluster register, catalog, deploy, resources (full console parity deliberately **not** a v1 goal) | S |
| Self-service secrets store onboarding (my own Vault/AWS SM paths) | S (v1.x) |
| Preview/ephemeral environments | S (v1.1) |

### 7.3 Incorporated additions (research-backed) — all now included in the v1 scope

All ten recommendations below are committed: P0/P1 items are in §7.1/§7.2 as **M**, the rest as **S** with their target v1.x release. The numbered list keeps the evidence trail.

1. **Audit logging (P0)** — table stakes in every commercial IDP, painful in OSS Backstage[^17^]; a wedge for Inari. Immutable events for every action incl. agent syncs.
2. **Approval workflows (P0)** — Port ships them natively[^17^], Kratix added them in 2026[^18^]; they are the governance boundary in a multi-tenant platform.
3. **ESO/secrets integration (P0)** — namespaced SecretStores per tenant[^19^]; "secrets sync" as a discoverable catalog capability.
4. **RBAC mapping UX (P1, differentiator)** — the Keycloak-groups→cluster-roles mapping is a notorious pain point (KubeVela built multi-cluster authz over exactly this)[^20^].
5. **Scaffolding/software templates (P1)** — the canonical first adoption win (Backstage Scaffolder proves the pattern)[^21^]; bind tenant context (namespace, group, RBAC) into every scaffold.
6. **DNS/ingress/TLS automation (P1)** — ExternalDNS + cert-manager as platform-cluster shared services, surfaced as catalog capabilities; "basic DNS" is in the canonical MVP definition.[^22^]
7. **Notifications (P2)** — cheap, high perceived polish.
8. **Scorecards from capability data (P2→v1.1)** — score what clusters *actually have*; this is where capability discovery compounds.
9. **Adoption metrics built-in (P2)** — ~45% of platform teams measure nothing[^23^]; built-in dashboards serve both users and the project's credibility.
10. **Per-tenant cost visibility (P2, v1.2)** — almost no IDP does FinOps; even basic cost awareness prevents over-provisioning blowback.

**Explicitly out of v1** (per MVP evidence): multi-cloud beyond AWS, service mesh, advanced CI/CD (blue/green), deep observability stacks, high-compliance features, Control Tower/AFT-style vending governance (Inari-native zone vending ships instead, §5.12).

---

## 8. UI Outline

![UI outline and information architecture](diagrams/d5-ui-architecture.png)

### 8.1 Approach

- **React + TypeScript + Vite** SPA served by `inari-server`; Tailwind + shadcn/ui component system; light, warm-neutral theme (dark mode supported).
- **Module Federation host**: first-party pages are built as internal remotes on the same SDK as third-party extensions — the shell only composes.
- **Schema-driven forms everywhere**: the deploy wizard walks OpenAPI v3 schemas (RJSF core) + OLM/KubeVela-style UI hints + platform policy (locked fields, defaults).
- **Tenant context switcher** is a first-class chrome element (org → team scope), reflecting the multi-tenant core. **Multi-org UX (decided):** users belonging to several tenants get a global **"All tenants" home** (aggregated resources, pending approvals, notifications across orgs), a fast switcher with recents, strict per-org scoping once a tenant is selected, and deep links that carry tenant context.
- Design tokens + blueprints defined in `inari-ui-plugin-sdk`; extension dev harness runs a remote standalone with a mocked control plane.

### 8.2 Information architecture (navigation)

```
├── Overview (dashboard: my tenants, recent resources, pending approvals, cluster health)
├── Catalog
│    ├── Browse (filters: cluster compatibility, category, source: curated/discovered/platform)
│    └── Item detail (docs, versions, schema preview, policy, "Deploy" CTA)
├── Deploys / Resources
│    ├── Resource instances (cross-cluster inventory, health, owner)
│    └── Instance detail (status, composed resources, ArgoCD link, actions)
├── Clusters
│    ├── List + register wizard (token, install manifest)
│    └── Detail (capabilities, installed providers, connection health, RBAC mapping tab)
├── Fleet (ClusterSets · staged rollouts · drift · agent upgrades)
├── Cloud Accounts (register AWS, validation status, ProviderConfigs)
├── Platform (platform-cluster apps, platform-scoped resources: realms/clients/DNS)
├── Tenant Zones (vend-new-zone wizard · zone lifecycle · decommission)
├── Templates / Scaffolds
├── Approvals (inbox + requested)
├── Audit Log
├── Extensions (installed, marketplace-ish registry, add remote)
└── Settings
     ├── Tenant/Org (members, teams, IdP brokering, domains)
     ├── Identity (clients, scopes, RBAC mapping)
     ├── Policies (packs, visibility, git, approvals, exemptions · compliance)
     └── Tokens & Secrets (registration tokens, ESO stores)
```

### 8.3 Key screens (v1 build order)

1. **Login / tenant select** (OIDC, org switcher)
2. **Cluster list + register wizard** — the first "aha": token → manifest → cluster comes online
3. **Cluster detail: Capabilities tab** — live, discovered catalog of *this* cluster (the differentiator, showcased)
4. **Catalog browse + item detail**
5. **Deploy wizard** (schema form → review → GitOps PR/commit → live status)
6. **Resources inventory + instance detail**
7. **Cloud account registration wizard** (CloudFormation snippet walkthrough)
8. **RBAC mapping** (Keycloak groups ↔ cluster roles, visual matrix)
9. **Approvals inbox / audit log** (simple, dense tables)
10. **Platform page** (platform apps + tenant platform resources)
11. **Fleet & rollout detail** (ClusterSets, stage progress, per-cluster status, gate approvals, drift)

### 8.4 Extension slots (v1)

| Slot | Example content |
|---|---|
| `NavItem` | Extension entry in sidebar |
| `CatalogCard` | Extra badges/actions on catalog items (e.g., cost estimate) |
| `ClusterTab` | ArgoCD health tab, observability deep-links |
| `InstanceAction` | ArgoCD sync/rollback/refresh, custom Lua actions |
| `FormWidget` | Custom field widgets (secret pickers, AWS resource pickers) |
| `Page` | Full extension pages (behind extension RBAC) |

---

## 9. Development Plan

Assumptions: 4–6 engineers, polyrepo, 2-week iterations. Milestones are capability-gated, not date-gated; weeks are indicative for a 5-engineer team (~30 weeks to v1.0).

### M0 — Foundations (wks 1–5)
- Repo scaffolding (server, agent, ui, api, charts), CI/CD, signing, dev env (kind-based local platform cluster)
- **Day-0 bootstrap:** scripted first-platform-cluster install via `inari-helm-charts` (Inari never requires Inari to install); **backup/restore runbook** for PostgreSQL, OpenFGA store, Keycloak config, NATS
- Keycloak integration: `inari` realm, Organizations, console SSO login, tenant switcher
- Core data model + tenancy service; audit outbox; **OpenFGA** on the platform cluster (authorization model v1, `Authorizer` interface, tuple writer fed by the outbox)
- **Exit:** log in via OIDC, create org/teams, OpenFGA checks enforced on all API routes, empty shell UI with tenant context, **control plane restored from backup in a DR drill**

### M1 — Agent & Clusters (wks 4–11)
- `inari-api` protobuf: EventStream, registration, capability events
- Registration flow (one-time token → per-cluster OIDC client; ESO delivery)
- Agent: connect/heartbeat/resync; capability discovery watches (CRDs, OLM, XRDs)
- **Spike (ex-12.2/6):** OpenFGA performance at target scale — Check/ListObjects p99, tuple volume, PEP caching strategy
- **Spike (ex-12.2/7):** KRO `v1alpha1` upgrade drill — simulate an RGD API break; verify `CatalogItem` abstraction shields tenants + raw-CRD fallback
- **Spike (ex-12.2/8):** bundle-managed ArgoCD lifecycle — version-upgrade path; BYO detection/adoption flow
- Cluster registry UI: list, register wizard, detail with live capabilities
- **Exit:** kind tenant cluster registers, capabilities stream into the console; **all three spike reports delivered with go/no-go recommendations**

### M2 — Catalog & First Deploy (wks 10–17)
- Catalog service (normalization, visibility policies); RJSF deploy wizard
- Orchestrator: RGD instance render → tenant Git → tenant-local ArgoCD Application; status stream back
- `inari-catalog` seed: 4–6 golden-path KRO packages (namespace-as-a-service, web-service+DNS+TLS, s3-backed app)
- Resources inventory + instance detail; resource version upgrade flow (badge, diff preview)
- **Exit:** end-to-end golden path: register cluster → browse catalog → deploy → watch health in console

### M3 — Cloud, Platform Cluster & Governance (wks 16–23)
- AWS account onboarding (web-identity role, validation, ProviderConfig materialization) + Crossplane-based packages (PostgreSQL/S3)
- Platform cluster + `inari-operator` (Keycloak realm/client resources, DNS via ExternalDNS, tenant namespaces); platform apps catalog (Keycloak, cert-manager, ESO, ArgoCD)
- Approvals, RBAC mapping UX, notifications; impersonation for automation
- Policy Service: request-time OPA checks, render-time manifest checks; policy pack (Kyverno/CEL VAP) distribution via agent
- Cluster lifecycle states (cordon; decommission with drain + OIDC client revocation)
- Tenant Zone Factory: management-account connection, `tenant-zone-aws` RGD, end-to-end vend → provision → register flow incl. decommission path
- **Exit:** tenant self-serves a Keycloak realm + AWS Postgres; a new Tenant Zone can be vended from the console; approvals and policy guardrails enforced; full audit trail

### M4 — Extensibility, Templates & Hardening (wks 22–30)
- `inari-plugin-sdk` + extension host + `/api/extensions/*` proxy; `inari-ext-argocd` reference extension
- UI Module Federation remotes + blueprint slots; `inari-ui-plugin-sdk` + dev harness
- Scaffolding/templates; CLI v1; docs site; scorecards v1 (capability-fed readiness rules)
- Fleet Manager: ClusterSets, staged rollouts (canary/waves, health + approval gates, rollback), drift detection (report-only), agent upgrade channels, policy compliance view
- Security review (threat model per trust zone), load test against the **v1 scale envelope** (100 clusters, 5k resource instances, 50 concurrent agent streams, agent churn), upgrade/downgrade drills, fleet rollout game-day
- **Exit:** v1.0 — a third party can write an extension; a pilot team runs production workloads

### v1.x committed schedule (all recommended additions included)
- **v1.1** — scorecards (capability-fed readiness rules), environment lifecycle/TTL, preview/ephemeral environments
- **v1.2** — per-tenant cost visibility (AWS CUR/OpenCost), adoption/usage dashboards, observability deep-links, self-service secrets-store onboarding
- **Beyond** — multi-cloud providers, Control Tower/AFT coexistence

### Workstream swimlanes (parallelizable)
- **WS-Control** (2 eng): server modules, tenancy, catalog, orchestrator
- **WS-Agent** (1–2 eng): agent, protocol, GitOps integration
- **WS-UI** (1–2 eng): console, design system, schema forms, extension slots
- **WS-Content** (0.5–1 eng): catalog packages, charts, docs

---

## 10. Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Scope creep (most-cited IDP killer) | v1 never ships | Milestone exit gates; defer list enforced; pilot tenant from M2 |
| KRO API maturity (`v1alpha1`) | Breaking changes in packaging format | Isolate behind `CatalogItem` abstraction; pin kro versions; track kro's EKS-managed-capability trajectory |
| Catalog rot | Differentiator lost | Catalog is machine-generated by design; curated packages are versioned OCI artifacts with CI tests |
| Multi-tenant lifecycle bugs (Kratix postmortem: shared-namespace deletion wiped co-tenant)[^18^] | Cross-tenant data loss | Ownership semantics in `ResourceInstance`; admission policies; no shared tenant namespaces by default; chaos/e2e tests for teardown |
| Fleet-wide change blast radius (bad provider/policy/agent version rolled to every cluster) | Multi-tenant outage | Staged rollouts: canary ClusterSet first, health + approval gates, rollback-by-snapshot, cordon; policy packs and agents tested in catalog/contract CI before any wave |
| OpenFGA dual-write drift (tuples out of sync with resource DB) | Authorization incorrectness | Outbox-driven tuple writer + periodic reconciler re-deriving tuples from PostgreSQL; contract tests on every lifecycle event |
| Tenant-zone vending quotas & long async AWS ops (CreateAccount/CloseAccount limits) | Stuck/zombie zones | Quota pre-flight checks, resumable idempotent state machine, sub-resource status tracking, manual-intervention path |
| Realm-per-tenant temptation | IdP collapse at scale | Organizations for tenancy (GA KC 26); tenant realms only as workload-federation resources |
| Agent protocol churn | Cross-repo breakage | `inari-api` contract repo + versioned protos + contract CI |
| Bootstrap/registration token leakage | Rogue cluster enrollment | One-time + TTL'd tokens; optional double opt-in approval; cert/client revocation path |
| Backstage-comparison pressure ("why not plugins?") | Positioning confusion | Position as k8s-native, tenant-aware control plane, not a portal framework; optionally ship a read-only Backstage plugin later |
| Team competency spread (Go + React + k8s + Keycloak + Crossplane) | Velocity drag | Swimlane ownership; ADRs; extension SDKs keep domains decoupled |

## 11. Decisions Log (open questions — resolved)

| # | Question | Decision | Applied in |
|---|---|---|---|
| 1 | GitOps: bring-your-own vs bundled? | **Bundle-managed by default** — the agent installs and lifecycle-manages tenant-local ArgoCD as part of the tenant-zone baseline; **BYO flag** adopts an existing installation (version-skew policy documented) | §5.3 |
| 2 | Tenant Git model | **Platform-owned `<tenant>-inari-state` repo per tenant** holds rendered instances; application repos untouched; PR-vs-direct-commit stays a per-tenant policy | §5.3 |
| 3 | Licensing/governance | **Fully open source (Apache-2.0), no open-core gating** — sell services (consulting, support, managed ops), not product | §1 |
| 4 | CLI surface vs console parity | Full parity is **not** a v1 goal; CLI covers core flows only (login, cluster register, catalog, deploy, resources) | §7.2 |
| 5 | Agent upgrade contract | Control plane supports agent versions **N and N−1**, enforced by contract CI; channel-based auto-upgrades (`stable`/`canary`) | §5.11 |
| 6 | Multi-org users UX | Global **"All tenants" home** (aggregated resources, approvals, notifications), fast switcher with recents, strict per-org scoping once selected, deep links carry tenant context | §8.1 |
| 7 | Tenant zone tiers/quota | Single **starter tier** at v1 (one region, minimal managed node group, e.g. 2× t3.large); request the **minimum AWS account-creation quota**; more tiers with demand | §5.12 |

## 12. Pre-Implementation Agenda — resolve, explore, watch

Topics to settle *before* or *during* early implementation. Nothing here invalidates the architecture; leaving them unaddressed would surface as mid-flight rework.

### 12.1 Resolve before M0 (blocking decisions)

**Status: all five recommendations applied** — baked into §5.3 (git auth, brownfield modes, footprint budget, network stance), §5.9 (data model), and §9 (M0 bootstrap/DR, M4 scale envelope). The table remains as the decision record.

| # | Topic | Why it blocks | Recommendation |
|---|---|---|---|
| 1 | **Day-0 bootstrap & DR** ("who watches the watcher"): how the first platform cluster + control plane is installed; backup/restore of PostgreSQL, OpenFGA store, Keycloak config, NATS | Inari must not require Inari to install; no tenant onboards before restore is tested | `inari-helm-charts` + documented bootstrap script; tested backup/restore runbook as an M0 exit criterion |
| 2 | **Git provider matrix & git auth**: GitHub-only at v1? GitLab/self-hosted timing? Where agent git-write credentials live | Touches `inari-api` contracts and the agent | GitHub first (GitHub App credentials delivered via ESO — never PATs); GitLab in v1.x; provider abstraction in the orchestrator from day one |
| 3 | **Brownfield adoption semantics**: what happens to pre-existing ArgoCD/operators/resources when a cluster registers | Undecided semantics become data-destroying edge cases | Three modes per resource: adopt (under management), observe-only (inventory, no mutation), ignore; default = observe-only |
| 4 | **Agent footprint budget & v1 scale envelope** | Starter-tier EKS nodes are small; M4 load test needs targets | Agent ≤ 100m CPU / 128Mi; control plane tested at 100 clusters / 5k resource instances / 50 concurrent agent streams |
| 5 | **Network constraints stance**: corporate egress proxies, TLS inspection, air-gapped | gRPC streams through intercepting proxies fail in surprising ways | v1: document egress-only support + proxy guidance; WebSocket fallback and air-gapped support deferred with stated triggers |

### 12.2 Research / spike during M0–M1

| # | Topic | Question to answer |
|---|---|---|
| 6 | **OpenFGA performance** — **scheduled: M1 exit spike** | Check/ListObjects p99 latency and tuple volume at target scale; PEP caching strategy |
| 7 | **KRO `v1alpha1` upgrade drill** — **scheduled: M1 exit spike** | Simulate an RGD API break; prove the `CatalogItem` abstraction shields tenants; confirm the raw-CRD fallback path |
| 8 | **Bundle-managed ArgoCD lifecycle** — **scheduled: M1 exit spike** | Version-upgrade path for agent-managed ArgoCD; BYO detection/adoption flow (Decision 1 consequence) |
| 9 | **Terraform/OpenTofu interop** | Do future catalog items need a TF execution path, or does KRO/Crossplane compose cleanly for brownfield estates? Informs orchestrator interfaces now, feature later |
| 10 | **Threat model workshop** | STRIDE per trust zone (platform, agent channel, tenant, AWS trust chain); schedule in M3 ahead of the M4 security review; pen-test before selling services |

### 12.3 Watchlist — deferred to v2 (explicit triggers)

None of the below is v1/v1.x work. Each item lists the trigger that pulls it into the v2 plan.

| # | Topic | Trigger to act |
|---|---|---|
| 11 | **Control-plane downtime posture as a product claim** — agents reconcile autonomously through partitions | Make it a tested, documented claim before using it in services marketing/SLOs |
| 12 | **Compliance & data residency** — tenants export metadata only; SOC2 timeline | Gates enterprise *sales*, not code — start early if services target enterprises |
| 13 | **Community hygiene** — "Inari" trademark/domain/GitHub-org availability, DCO vs CLA, SECURITY.md + disclosure policy, supported-versions policy | Before public launch; trivial now, painful after |
| 14 | **Accessibility (WCAG) & console i18n** | First enterprise/design-partner requirement |
| 15 | **Multi-region control plane / agent latency** | First tenant outside the primary region with latency SLOs |
| 16 | **Air-gapped catalog distribution** (OCI artifacts make this feasible) | First regulated/air-gapped prospect |

---

## 13. Key References (research base)

- Agent/registration patterns: argocd-agent docs (architecture, sync protocol, auth), Rancher Fleet registration internals, OCM double opt-in + gRPC registration, Lens relay agents
- Catalog/packaging: kro.run (RGD API), OLM CSV descriptors, KubeVela UI-schema, Crossplane XRDs, Headlamp kro plugin
- Extensibility: hashicorp/go-plugin, ArgoCD proxy + UI extensions, Backstage new frontend system, Module Federation
- IAM/OIDC: Keycloak Organizations (KC 26), realm-scaling benchmarks, k8s structured JWT auth (1.34), kubelogin, impersonation, OpenFGA/SpiceDB comparisons, ESO multi-tenancy patterns
- Market/lessons: Backstage/Port/Cortex comparisons, Roadie & Frontiers maintenance-burden data, platformengineering.org MVP guidance, Kratix shared-resource postmortem, Cycloid/Qovery feature sets
- Full inline citations: see `research/r1-idp-landscape.md`, `research/r2-technical-patterns.md`, `research/r3-iam-oidc-patterns.md`
