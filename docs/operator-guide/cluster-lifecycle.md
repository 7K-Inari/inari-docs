# Cluster lifecycle

Registered clusters move through explicit lifecycle states (plan §5.11). Operators drive transitions from the cluster detail page or the fleet dashboard; every transition is audited.

## States

```
Pending → Active → Degraded → Cordoned → Decommissioned
```

| State | Meaning | Effect |
|---|---|---|
| `Pending` | Registered, token issued, agent not yet connected | No catalog visibility; registration token TTL ticking |
| `Active` | Agent connected, capabilities streaming | Full catalog + deploys available per tenant policy |
| `Degraded` | Agent disconnected or failing health | Deploys blocked on health gates; existing workloads keep reconciling autonomously |
| `Cordoned` | Operator-initiated freeze | New deploys blocked; existing workloads keep running; rollouts skip the cluster |
| `Decommissioned` | Terminal | Identity revoked, audit archived, record retained read-only |

`Degraded` is entered automatically (missed heartbeats / unhealthy agent status) and exits automatically on healthy reconnect (checksum resync closes any status gap).

## Cordoning

Cordon a cluster before maintenance windows, when pulling it out of rollout rotation, or as the first decommission step. Cordon never touches running workloads — it only blocks *new* desired state.

## Decommissioning

Decommission drains Inari-managed resources and revokes identity, in order:

1. **Cordon** — block new deploys.
2. **Drain** — Inari-managed `ResourceInstance`s are torn down in reverse-dependency order, **ownership-checked**: only resources Inari manages (managementMode `adopt` or platform-created) are touched; `observe-only` resources are left running by design. No shared tenant namespaces by default — teardown cannot wipe a co-tenant (plan §10).
3. **Revoke identity** — the cluster's Keycloak OIDC client (`cluster-<id>`) is disabled; the registration token record is invalidated. Tenant-side, remove the agent manifest (or let the tenant do it).
4. **Archive audit** — the cluster's audit trail is exported and archived; the cluster record becomes read-only.

Decommission is approval-gated when the cluster hosts resources owned by multiple teams.

:::warning Irreversibility
Decommission is terminal. Re-onboarding the same cluster is a fresh registration (new token, new OIDC client, new brownfield classification pass).
:::

## Related

- [Fleet rollouts](fleet-rollouts.md) — cordon interacts with rollout targeting
- [Tenant Zone vending](tenant-zones.md) — zone decommission wraps cluster decommission with account closure
- [Threat model](../security/threat-model.md) — tenant-zone teardown threat (Kratix postmortem) and its mitigations
