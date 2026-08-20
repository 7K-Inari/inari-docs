# UI extensions

The console is a **Module Federation host**: first-party pages are internal remotes on the same SDK offered to you, and third-party extensions register as runtime remotes via a manifest-driven registry (`remoteEntry.js` served via the backend/OCI) (plan §5.8, §8).

## The blueprint contract

You do not render free-form into the console. You register against **typed extension-point blueprints** (Backstage new-frontend style), declared in `inari-ui-plugin-sdk`:

| Slot | What you provide | Example |
|---|---|---|
| `NavItemBlueprint` | Sidebar entry | "ArgoCD" nav item |
| `CatalogCardBlueprint` | Badges/actions on catalog items | Cost estimate badge |
| `ClusterTabBlueprint` | Extra tab on cluster detail | ArgoCD health tab, observability deep-links |
| `InstanceActionBlueprint` | Actions on resource instances | Sync / refresh / rollback, custom Lua actions |
| `FormWidgetBlueprint` | Custom field widgets for the deploy wizard | Secret picker, AWS resource picker |
| `PageBlueprint` | Full pages (behind extension RBAC) | Extension dashboard |

Shared React singletons mean your remote uses the host's React, router, and design system (Tailwind + shadcn/ui + design tokens from the SDK) — do not bundle your own React.

## Project layout

```bash
inari extension init --ui-only my-cards
```

```
my-cards/
  src/
    index.ts          # registers blueprints with the host API
    CatalogCard.tsx
  package.json        # module-federation config generated for you
  plugin.yaml         # extension manifest (name, version, slots used)
```

## Dev harness

`inari-ui-plugin-sdk` ships a **dev harness** that runs your remote standalone against a mocked control plane:

```bash
inari extension dev    # serves remoteEntry.js + mock host with fixtures
```

Fixtures cover tenant context, cluster/resource shapes, and the host APIs (navigation, toast, current user, auth context). Test every blueprint slot you register against the harness before publishing.

## Backend pairing

A UI extension that needs data beyond host APIs pairs with a [backend extension](backend-extensions.md): the same `<name>` exposes `/api/extensions/<name>/*`, and the SDK's fetch helper calls it with the user's auth context applied. See `inari-ext-argocd`, which pairs its `InstanceAction`/`ClusterTab` remotes with a backend plugin that proxies tenant-local ArgoCD actions.

## Runtime registration

Installed extensions are registered at runtime from the extension registry (**Extensions** page in the console). Remotes load lazily; a failing remote is isolated to its slot and does not break the shell.
