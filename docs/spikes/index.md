# Research Spikes

Time-boxed research spikes that gate milestone exits. Each report ends with a go/no-go recommendation and the impact on the platform plan if no-go.

## M1 exit spikes (plan §12.2 items 6–8)

| Spike | Report | Track | Recommendation |
|---|---|---|---|
| OpenFGA performance at target scale | [m1-openfga-performance](./m1-openfga-performance.md) | A — real execution (local OpenFGA v1.18.3 + synthetic tuple load) | **GO-WITH-CONDITIONS** |
| KRO `v1alpha1` upgrade drill | [m1-kro-upgrade-drill](./m1-kro-upgrade-drill.md) | B — reproducible harness, preliminary findings | **GO-WITH-CONDITIONS** |
| Bundle-managed ArgoCD lifecycle | [m1-argocd-bundle-lifecycle](./m1-argocd-bundle-lifecycle.md) | B — reproducible harness, preliminary findings | **GO-WITH-CONDITIONS** |

## Re-running the experiments

All harnesses live under `harness/`:

- `harness/openfga/` — Go harness: authorization model (`model.json`), synthetic tuple generator at the v1 scale envelope and 10×, Check/ListObjects benchmark. Requires only a local `openfga` binary.
- `harness/kro/run.sh` — kind + kro upgrade drill (requires docker, kind, kubectl, helm).
- `harness/argocd/run-upgrade-drill.sh` and `detect-byo.sh` — bundle upgrade/rollback drill and BYO detection probe (requires docker, kind, kubectl).

Track B reports contain figures marked **PRELIMINARY**; they become final when the harness is re-run on a docker-capable machine and the artifacts are attached to the report.
