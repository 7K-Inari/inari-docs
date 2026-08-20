# Extension Authors

Documentation for engineers building **Inari extensions**: backend plugins (gRPC sidecar contract) and UI extensions (Module Federation remotes) (plan §5.8).

First-party features eat the same dogfood — the ArgoCD action extension and AWS provider support are built on the public SDKs. The canonical reference implementation is **`inari-ext-argocd`** (`github.com/7K-Inari/inari-ext-argocd`): read it alongside these guides.

## Contents

- [Backend extensions](backend-extensions.md) — the versioned gRPC plugin contract, lifecycle, and the authenticated proxy path
- [UI extensions](ui-extensions.md) — Module Federation remotes, blueprint extension points, dev harness
- [Packaging & governance](packaging-governance.md) — signing, versioning, registry, and extension governance

## Quickstart

```bash
inari extension init my-extension          # scaffolds backend + UI skeleton
cd my-extension
inari extension dev                        # runs against the dev control-plane harness
```

Then follow the tutorial: [write your first extension](../tutorials/first-extension.md).
