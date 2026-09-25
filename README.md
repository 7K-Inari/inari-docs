# inari-docs

Documentation for the **Inari** multi-tenant Internal Developer Platform: docs site, ADRs, user/operator/extension-author guides, tutorials.

**Start here:** [docs/architecture/inari-platform-plan.md](docs/architecture/inari-platform-plan.md) — the canonical architecture & development plan (repo topology, milestones M0–M4, V1 feature set, decisions log).

## Quickstart — writing docs locally

The site is built with [MkDocs](https://www.mkdocs.org/) ([Material theme](https://squidfunk.github.io/mkdocs-material/)). **All content lives in `docs/`** — that folder *is* the docs root, so you edit markdown in place and it shows up on the site.

```bash
pip install -r requirements-docs.txt   # first time only (use a venv)
mkdocs serve                           # dev server at http://localhost:8000 with hot reload
```

Other commands:

```bash
mkdocs build          # static build into site/ (same as CI)
mkdocs build --strict # what CI runs — warnings (broken links, nav gaps) fail the build
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

Section labels and ordering are controlled by `.pages` files in each folder (and the root `docs/.pages`) — you don't need to touch `mkdocs.yml` when adding pages.

## CI & deployment

- **PRs** (`.github/workflows/docs-ci.yml`): markdownlint on `docs/` plus a strict MkDocs build (`mkdocs build --strict`).
- **Push to `main`** (`.github/workflows/docs-release.yml`): strict build, then the `site/` output is published to the org docs S3 bucket under the `inari/` prefix via the shared `release-docs.yml` reusable workflow.

Publishing requires these repo secrets: `DOCS_S3_ACCESS_KEY_ID`, `DOCS_S3_SECRET_ACCESS_KEY`, `APPDOCS_S3_BUCKET`, `DOCS_S3_ENDPOINT_URL`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) — including when to write an ADR.
