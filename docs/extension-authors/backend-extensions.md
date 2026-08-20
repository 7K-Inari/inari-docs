# Backend extensions

Backend extensions are plugins that run beside `inari-server` over a **versioned gRPC contract** (`inari-plugin-sdk`), in the hashicorp go-plugin style: handshake cookie, protocol version, checksum verification, crash isolation (plan §5.8). Use cases: ArgoCD actions, custom capability detectors, external inventory importers.

## The contract

- The protobuf contract lives in `inari-api` (`plugin/v1`). Server, SDK, and plugins pin its versions; contract CI guards compatibility.
- A plugin declares its **name, version, and protocol version** at handshake. Mismatched protocol versions are refused; the extension host runs plugins as sidecar containers/subprocesses so a crash never takes down the control plane.
- The SDK (`inari-plugin-sdk`, Go) handles the handshake, lifecycle (init/health/shutdown), and passes an **auth context** on every call: the authenticated user, their tenant, and their OpenFGA-checked permissions. You never see raw tokens.

## How requests reach your plugin

Plugin HTTP endpoints surface through the control plane's authenticated reverse-proxy path (the ArgoCD proxy-extension pattern):

```
POST /api/extensions/<name>/<your-path>
       │  1. OIDC JWT validated (gateway)
       │  2. OpenFGA check: extensions, invoke, <name>
       │  3. Sensitive headers stripped (Authorization, cookies)
       ▼
   your plugin's gRPC/HTTP handler
```

Implications:

- **You do not do your own authn.** Treat the SDK-provided auth context as authoritative.
- Per-invoke RBAC (`extensions, invoke, <name>`) is enforced before traffic reaches you; finer-grained checks inside your plugin use the SDK's `Authorizer` helper.
- Your plugin runs with no cloud or cluster credentials. If it needs to act on a tenant cluster, it does so through the SDK's dispatch helpers (desired-state commands over the agent channel) — never via stored kubeconfigs.

## Building one

```bash
inari extension init --backend-only my-actions
```

The scaffold gives you a Go module with the SDK wired, a `plugin.yaml` manifest (name, version, protocol version, checksum), and a health endpoint. Implement your handlers, add them to the manifest's endpoint list, and test locally with `inari extension dev`, which runs a mocked control plane.

## Reference implementation

**`inari-ext-argocd`** is the first-party ArgoCD extension and the reference for everything above: sync/refresh/rollback plus custom Lua resource actions exposed as `InstanceAction`s, and status surfacing for the UI. If you are unsure how to structure a plugin, mirror its layout.

## Versioning

- Bump your plugin's semver independently; the manifest pins the **protocol version** you speak.
- The control plane refuses plugins whose protocol version it does not support — test upgrades against the dev harness before publishing.
- Distribute as a container image; checksums are verified at load time (see [packaging & governance](packaging-governance.md)).
