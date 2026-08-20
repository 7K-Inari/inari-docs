# inari-docs

Documentation for the **Inari** multi-tenant Internal Developer Platform: docs site, ADRs, user/operator/extension-author guides, tutorials.

**Start here:** [docs/architecture/inari-platform-plan.md](docs/architecture/inari-platform-plan.md) — the canonical architecture & development plan (repo topology, milestones M0–M4, V1 feature set, decisions log).

## Quickstart — writing docs locally

The site is built with [Docusaurus](https://docusaurus.io/). **All content lives in `docs/`** — that folder *is* the docs root, so you edit markdown in place and it shows up on the site.

```bash
npm ci          # first time only
npm start       # dev server at http://localhost:3000 with hot reload
```

Other commands:

```bash
npm run build   # static build into build/ (same as CI)
npm run serve   # serve the production build locally
```

### Where things go

| Path | Content |
|---|---|
| `docs/architecture/` | Architecture & design docs (the canonical plan lives here) |
| `docs/user-guide/` | Developer-facing guides |
| `docs/operator-guide/` | Platform operator runbooks & guides |
| `docs/extension-authors/` | Plugin/UI-extension author docs |
| `docs/tutorials/` | Step-by-step walkthroughs |
| `docs/security/` | Security docs: threat model per trust zone, review artifacts |
| `docs/adr/` | Architecture Decision Records (see [CONTRIBUTING.md](CONTRIBUTING.md)) |

Sidebar labels and ordering are controlled by `_category_.json` files in each folder — you don't need to touch `sidebars.ts` when adding pages.

## CI & deployment

`.github/workflows/deploy.yml` builds the site on every PR and push to `main`, and deploys `build/` to **GitHub Pages** via the official `actions/deploy-pages` action (site: <https://7k-inari.github.io/inari-docs/>).

If GitHub Pages is not enabled on the repo, the build job still runs and passes, and the built site is available as a downloadable workflow artifact — only the `deploy` job requires Pages. To enable Pages: repo **Settings → Pages → Source: GitHub Actions**.

Using a custom domain or different org? Update `url`/`baseUrl` in `docusaurus.config.ts`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) — including when to write an ADR.
