#!/usr/bin/env bash
# KRO v1alpha1 upgrade-drill harness (spike: m1-kro-upgrade-drill).
# Requires: docker, kind, kubectl, helm (or curl for kro manifests).
# Usage: ./run.sh [kro_version_a] [kro_version_b]   e.g. ./run.sh 0.2.3 0.3.0
set -euo pipefail
KRO_A="${1:-0.2.3}"
KRO_B="${2:-0.3.0}"
CLUSTER="${CLUSTER_NAME:-kro-drill}"

echo "== kind cluster =="
kind create cluster --name "$CLUSTER" --wait 60s || kind get clusters | grep -q "$CLUSTER"

echo "== install kro $KRO_A =="
helm upgrade --install kro oci://ghcr.io/kro-run/kro/kro \
  --namespace kro --create-namespace --version "$KRO_A" --wait

echo "== apply RGDs (v1alpha1) =="
kubectl apply -f "$(dirname "$0")/rgd-namespace-as-a-service.yaml"
kubectl apply -f "$(dirname "$0")/rgd-web-service.yaml"
kubectl wait --for=condition=Ready resourcegraphdefinition namespace-as-a-service --timeout=120s

echo "== instantiate samples =="
kubectl apply -f "$(dirname "$0")/instances.yaml"
kubectl get crd | grep -E 'namespaceasaservice|webservice' || true

echo "== record pre-break state =="
kubectl get resourcegraphdefinitions -o yaml > pre-break-rgd.yaml
kubectl get crd -o yaml > pre-break-crds.yaml

echo "== SIMULATE API BREAK: upgrade kro $KRO_A -> $KRO_B =="
helm upgrade --install kro oci://ghcr.io/kro-run/kro/kro \
  --namespace kro --version "$KRO_B" --wait

echo "== post-break observations =="
kubectl get resourcegraphdefinitions -o yaml > post-break-rgd.yaml
kubectl get crd -o yaml > post-break-crds.yaml
diff pre-break-crds.yaml post-break-crds.yaml > crd-diff.txt || true
kubectl get events --sort-by=.lastTimestamp | tail -30 > post-break-events.txt || true

echo "== verify existing instances still reconcile =="
kubectl get -f "$(dirname "$0")/instances.yaml" -o wide || true

echo "== RAW-CRD FALLBACK: deploy same resources via generated CRD directly =="
kubectl apply -f "$(dirname "$0")/fallback-raw-crd-instance.yaml"

echo "Done. Inspect pre/post artifacts in $(pwd)."
