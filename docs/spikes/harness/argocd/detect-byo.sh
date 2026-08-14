#!/usr/bin/env bash
# BYO ArgoCD detection/adoption probe — the logic inari-agent will run on registration.
# Classifies any pre-existing ArgoCD and prints an adoption decision per the
# version-skew policy documented in docs/spikes/m1-argocd-bundle-lifecycle.md.
# Requires: kubectl context pointed at the candidate cluster.
set -euo pipefail

SUPPORTED_MIN="${SUPPORTED_MIN:-v2.14.0}"   # policy: agent supports ArgoCD N and N-1 minor lines
SUPPORTED_MAX="${SUPPORTED_MAX:-v3.99.0}"

echo "== 1. locate argocd-server deployment across namespaces =="
DEPLOYS=$(kubectl get deploy -A -o json | jq -r '.items[] | select(.metadata.name=="argocd-server") | [.metadata.namespace, .spec.template.spec.containers[0].image] | @tsv')
if [ -z "$DEPLOYS" ]; then
  echo "RESULT: no ArgoCD found -> bundle-managed install path (default)"
  exit 0
fi

echo "$DEPLOYS" | while read -r ns image; do
  echo "== 2. found argocd-server in namespace '$ns', image '$image' =="
  version=$(echo "$image" | sed -E 's/.*:(v?[0-9]+\.[0-9]+\.[0-9]+).*/\1/')
  echo "   detected version: $version"

  echo "== 3. install method heuristics =="
  helm_secret=$(kubectl -n "$ns" get secret -l owner=helm -o name 2>/dev/null | grep -i argocd | head -1 || true)
  if [ -n "$helm_secret" ]; then method="helm"; else method="plain-manifests"; fi
  echo "   install method: $method"

  echo "== 4. existing state to capture (never mutated on observe-only) =="
  kubectl -n "$ns" get applications.argoproj.io --no-headers 2>/dev/null | wc -l | xargs echo "   Applications:"
  kubectl -n "$ns" get appprojects.argoproj.io --no-headers 2>/dev/null | wc -l | xargs echo "   AppProjects:"

  echo "== 5. skew classification =="
  if [[ "$version" > "$SUPPORTED_MIN" || "$version" == "$SUPPORTED_MIN" ]]; then
    echo "RESULT: BYO candidate — version $version within supported window [$SUPPORTED_MIN, $SUPPORTED_MAX]"
    echo "        default managementMode=observe-only; adoption requires explicit, audited opt-in per §12.1/3"
  else
    echo "RESULT: BYO candidate OUT OF SKEW ($version < $SUPPORTED_MIN) -> refuse adoption; offer bundle-managed side-by-side or document tenant upgrade"
  fi
done
