# Backup & restore (DR runbook)

The control plane is recoverable from backups alone. Restorability is an M0 exit criterion: **no tenant onboards before a restore drill has passed** (plan §12.1/1).

## What must be backed up

| Component | Contents | Method |
|---|---|---|
| PostgreSQL | Control-plane state: tenants, clusters, catalog, resource instances, audit events, outbox | Scheduled `pg_dump` (or WAL + base backup for PITR) to object storage |
| OpenFGA store | Authorization model + relationship tuples | Store export; **also re-derivable** — the periodic reconciler re-derives tuples from PostgreSQL, so a tuple restore is a fallback, not a requirement |
| Keycloak config | `inari` realm, Organizations, clients, IdP brokering, group mappings | Realm export (JSON) + Keycloak DB backup; per-cluster client secrets are rotatable, not backed up |
| NATS | Internal event bus | Transient by design — no backup; the outbox in PostgreSQL is the source of truth for event replay |

Backup jobs are installed by the platform baseline chart and write to the object-storage bucket configured at bootstrap. Verify backups exist daily; alert on missing/failed backup runs.

## Restore procedure

1. **Stand up a clean platform cluster** — run [day-0 bootstrap](bootstrap.md) with the same hostnames/OIDC issuer.
2. **Stop writers** — scale `inari-server` and `inari-operator` to 0.
3. **Restore PostgreSQL** — restore the latest dump (or PITR to just before the incident). Confirm the outbox table is present.
4. **Restore Keycloak** — restore its database, then re-import the realm export if the DB restore predates it. Verify the `inari` realm, Organizations, and per-cluster OIDC clients exist.
5. **Restore OpenFGA** — import the latest store export **or** load the authorization model and let the tuple reconciler re-derive tuples from PostgreSQL (slower; safe).
6. **Restart the control plane** — scale `inari-server` back up. Agents reconnect automatically (backoff + checksum resync); desired state converges without re-registration.
7. **Verify** — checklist below.

## Post-restore verification checklist

- [ ] OIDC login works; tenant switcher shows all organizations
- [ ] OpenFGA `Check` succeeds for a known tenant admin and fails for a non-member
- [ ] All clusters show connected agents (allow for resync lag)
- [ ] Audit log queryable for pre-incident events
- [ ] A no-op deploy request renders and reaches tenant Git (end-to-end smoke)
- [ ] Backup jobs resumed against the new cluster

## Drill cadence

A full restore drill runs at least once per milestone and after any change to the backup jobs. Drill results are recorded in the ops section (see the [upgrade/downgrade drill](../ops/upgrade-downgrade-drill.md) for the format).

:::note Tenant-side state
Tenant clusters keep reconciling autonomously through a control-plane outage (pull-never-push, desired-state design). A control-plane restore does not touch tenant workloads; agent resync reconciles any status gap on reconnect.
:::
