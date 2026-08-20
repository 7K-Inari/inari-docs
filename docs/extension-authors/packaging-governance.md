# Packaging & governance

How extensions are packaged, signed, distributed, and governed (plan §5.8, §5.10).

## Packaging

| Part | Artifact | Distribution |
|---|---|---|
| Backend plugin | Container image (go-plugin sidecar) | OCI registry, cosign-signed, checksum in `plugin.yaml` verified at load |
| UI remote | `remoteEntry.js` bundle | Served via the backend from an OCI artifact, same signing pipeline |
| Manifest | `plugin.yaml` (name, version, protocol/contract version, slots/endpoints, checksums) | Inside the OCI artifact |

Builds should produce SLSA provenance like all Inari release artifacts; unsigned or checksum-mismatched plugins are refused by the extension host.

## Registry & installation

- The console's **Extensions** page lists installed extensions and the available registry.
- Installing an extension is a platform-governed action: platform engineers approve new extensions (and version upgrades) before they become visible to tenants.
- Per-tenant visibility can be scoped like catalog items; tenants only see extensions their platform team has enabled for them.

## Governance rules for authors

1. **Inherit auth, never roll your own** — use the SDK auth context; plugins that ask users for credentials are rejected in review.
2. **No ambient credentials** — plugins run without cluster/cloud credentials; act through the SDK's dispatch helpers.
3. **Declare your slots and endpoints** in the manifest; undeclared surface area is not proxied.
4. **Pin the contract version** you speak and test upgrades against the dev harness; protocol mismatches fail closed.
5. **Crash isolation is not error isolation** — return structured errors; the host surfaces them in the console.

## Trust tiers

- **First-party** (`inari-ext-*`): built on the public SDKs, shipped by the Inari project, governed like core.
- **Third-party (reviewed)**: checksum/signature verified, reviewed by the platform team, installed per-tenant or fleet-wide.
- **Untrusted / tenant-supplied**: WASM-sandboxed extensions are reserved for this tier and are **not in v1** — do not build against an assumed WASM runtime (plan §5.8).

## Security considerations

Extensions sit inside the platform trust zone: their traffic crosses boundary B6 in the [threat model](../security/threat-model.md). The proxy strips sensitive headers, enforces per-extension RBAC, and the audit log records every invocation. Treat the manifest's declared endpoints as your attack surface and keep them minimal.
