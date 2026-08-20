# Tutorial: Write your first extension

Scaffold, build, and install a minimal Inari extension with a backend plugin and a UI remote. ~45 minutes.

**Prerequisites:** Go, Node.js, and the `inari` CLI; access to a dev control plane (or the bundled mock harness); a platform engineer to install your finished extension. Read [backend extensions](../extension-authors/backend-extensions.md) and [UI extensions](../extension-authors/ui-extensions.md) first, and keep **`inari-ext-argocd`** open as the reference.

## 1. Scaffold

```bash
inari extension init hello-ext
cd hello-ext
```

You get: a Go backend module with the SDK wired, a TS UI remote with Module Federation config, a `plugin.yaml` manifest, and a health endpoint.

## 2. Add a backend endpoint

Implement a handler in the Go module. The SDK hands you an auth context per call — use it, never raw tokens:

```go
func (p *Plugin) HandleHello(ctx context.Context, ac auth.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
    // ac.User, ac.Tenant, ac.Permissions are already authenticated/authorized
    return &pb.HelloResponse{Message: "hello " + ac.User.Name + " from " + ac.Tenant}, nil
}
```

Declare the endpoint in `plugin.yaml` under `endpoints:` so the proxy exposes it at `/api/extensions/hello-ext/hello`.

## 3. Add a UI slot

Register a `CatalogCardBlueprint` in `src/index.ts`:

```ts
import {CatalogCardBlueprint} from '@inari/ui-plugin-sdk';

export default [
  CatalogCardBlueprint.make({
    id: 'hello-ext.card',
    component: HelloCard, // calls /api/extensions/hello-ext/hello via the SDK fetch helper
  }),
];
```

## 4. Run the dev harness

```bash
inari extension dev
```

The harness serves your `remoteEntry.js`, runs your backend against a mocked control plane, and gives you fixtures for tenant context and catalog items. Iterate until the card renders and the endpoint answers.

## 5. Package

```bash
inari extension package   # builds the image + remote bundle, emits checksums into plugin.yaml
```

Sign the artifact with cosign like every Inari artifact; unsigned plugins are refused at load.

## 6. Install

Hand the OCI reference to a platform engineer (extension installation is platform-governed). They install it from **Extensions → Add remote**, approve it, and scope tenant visibility. Your card now appears on catalog items, and `/api/extensions/hello-ext/hello` answers for users with `extensions, invoke, hello-ext`.

## Where next

- Realistic backend: actions on tenant clusters via the SDK's dispatch helpers — mirror how `inari-ext-argocd` tunnels ArgoCD actions over the agent stream.
- More slots: `ClusterTab`, `InstanceAction`, `FormWidget`, `Page` ([blueprint reference](../extension-authors/ui-extensions.md)).
- Governance: [packaging & trust tiers](../extension-authors/packaging-governance.md).
