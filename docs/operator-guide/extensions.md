# Installing extensions

Extensions come in two kinds (plan §5.8), often paired under one name
(`spec.kinds: [backend, ui]` in the extension's `extension.yaml`):

- **Backend extensions** — sidecar plugins verified via the SDK handshake;
  their HTTP endpoints surface through `/api/extensions/<name>/*` behind the
  `extensions, invoke, <name>` RBAC verb.
- **UI extensions** — Module Federation remotes whose `remoteEntry.js` is
  **served by the control plane itself**, from the registered external HTTPS
  URL or from an OCI artifact. The console never fetches third-party origins.

Both are registered per tenant through the extension registry API (writes
require the `platform-engineer` relation, reads any `viewer`).

## Registering a backend extension

```bash
curl -X POST "$INARI/api/v1/tenants/$ORG/extensions" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{
        "name": "inari-ext-argocd",
        "version": "0.1.1",
        "kind": "backend",
        "endpoint": "http://inari-ext-argocd.extensions.svc:8080"
      }'

# Run the SDK handshake; the extension transitions pending → ready.
curl -X POST "$INARI/api/v1/tenants/$ORG/extensions/<id>/verify" \
  -H "Authorization: Bearer $TOKEN"
```

## Registering a UI extension

Use the `ui` block of the extension's `extension.yaml` as the source of truth
(remoteEntry URL, slots, permission):

```bash
curl -X POST "$INARI/api/v1/tenants/$ORG/extensions/ui" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{
        "name": "inari-ext-argocd",
        "version": "0.1.0",
        "remoteEntry": "https://github.com/7K-Inari/inari-ext-argocd/releases/download/ui-v0.1.0/remoteEntry.js",
        "slots": [
          {"kind": "cluster-tab", "name": "argocd-health"},
          {"kind": "catalog-card", "name": "argocd-badge"},
          {"kind": "instance-action", "name": "argocd-sync"},
          {"kind": "instance-action", "name": "argocd-refresh"},
          {"kind": "instance-action", "name": "argocd-rollback"}
        ],
        "requiredPermission": "extensions:invoke:inari-ext-argocd",
        "enabled": true
      }'
```

Notes:

- Exactly one of `remoteEntry` (external HTTPS URL) or `remoteEntryOci`
  (oras artifact ref carrying a `remoteEntry.js` layer) is set. `checksum`
  optionally pins the asset's sha256 (hex); fetches failing the pin are
  rejected.
- Re-POSTing the same `name` updates the descriptor and version (upsert) and
  invalidates the control plane's cached asset. If a backend extension with
  the same name already exists, the UI descriptor is paired onto that row.
- The registry response reports a **server-relative** `remoteEntryUrl`
  (`/api/v1/tenants/<org>/extensions/ui/<name>/remoteEntry.js`) — this is
  what the console loads; the registered upstream URL is never exposed to
  browsers.
- Listing: `GET /api/v1/tenants/$ORG/extensions/ui`. Removal:
  `DELETE /api/v1/tenants/$ORG/extensions/ui/<name>` (a paired backend row is
  kept; a UI-only row is deleted).
- Users without `requiredPermission` do not see the extension's slots: the
  console reads its effective invoke verbs from
  `GET /api/v1/tenants/<org>/authz/self/extensions` and hides denied remotes.
  The `remoteEntry.js` asset route itself is unauthenticated (the Module
  Federation runtime loads entries via `<script>` injection, which carries
  no `Authorization` header); the payload is public client-side JavaScript,
  integrity-pinned hub-side, and disabled extensions are not served.
- On upsert, omitting `enabled` preserves the stored value — re-registration
  never silently re-enables a disabled extension.

## Verifying the install

1. Open the console **Extensions** page for the tenant: the UI remote should
   show load state **ready** and the backend plugin **healthy**.
2. Slot contributions appear where the blueprints bind: e.g. the ArgoCD
   extension adds an **ArgoCD health** tab on cluster detail, a catalog-card
   badge, and **Sync / Refresh / Rollback** instance actions. A failing
   remote is isolated to its slot and never breaks the shell.

## Supply-chain verification (OCI sources)

Set `INARI_UI_EXTENSION_VERIFY=true` to require cosign keyless signatures on
`remoteEntryOci` artifacts before the control plane serves them, pinning the
signing workflow with `INARI_UI_EXTENSION_COSIGN_IDENTITY` /
`INARI_UI_EXTENSION_COSIGN_ISSUER` (same trust model as template OCI
ingestion, `INARI_SCAFFOLD_TEMPLATE_VERIFY`). HTTPS sources are guarded
(https-only, no cross-scheme redirects, 5 MiB cap) and should be
checksum-pinned when the release process stamps digests.
