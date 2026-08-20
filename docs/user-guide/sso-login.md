# SSO login & tenants

Inari is OIDC-first: every sign-in goes through the platform Keycloak (`inari` realm), and tenancy is a Keycloak Organization (plan §5.4).

## Signing in

1. Open the console URL for your platform and choose **Sign in**.
2. You are redirected to the platform SSO. If your tenant has a **brokered corporate IdP**, entering your work email routes you to it automatically (verified-domain routing).
3. First-time users accept an organization invitation; your teams and permissions appear immediately after.

Everything in the console and API is authorized by **token claims, never by URL or header**. Tokens carry your organization claim and group memberships; per-service audiences prevent token reuse across services.

## Tenant context

The **tenant switcher** in the top chrome selects your working organization (and team scope where relevant). Once a tenant is selected, scoping is strict: every list, deploy, and API call is tenant-scoped.

If you belong to several organizations (e.g. contractors), you get an **"All tenants" home** ([ADR-0008](../adr/0008-multi-org-ux-all-tenants-home.md)):

- Aggregated view of your resources, pending approvals, and notifications across orgs.
- The aggregate home is read-only — acting on anything takes you into that tenant's explicit context.
- Deep links carry tenant context; a link to a resource in a tenant you cannot access fails closed.

## Teams and permissions

Teams are Keycloak groups (`tenant-<slug>/<team>`). They drive both the in-console RBAC (OpenFGA relationships reference teams, never individuals) and Kubernetes RBAC mapping (see [kubectl access](kubectl-access.md)). Membership changes are managed by your tenant admin in Keycloak only.

## CLI login

```bash
inari login          # OIDC device flow — opens a browser, or paste a code
inari login --org acme
```

The CLI covers core flows only ([ADR-0006](../adr/0006-cli-core-flows-only.md)); anything beyond login, cluster register, catalog, deploy, and resources is done in the console.
