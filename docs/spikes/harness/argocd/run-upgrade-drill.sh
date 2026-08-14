#!/usr/bin/env bash
# Bundle-managed ArgoCD lifecycle harness (spike: m1-argocd-bundle-lifecycle).
# Requires: docker, kind, kubectl. Installs "bundle" ArgoCD vA, upgrades to vB,
# verifies Applications keep reconciling, then rolls back.
# Usage: ./run-upgrade-drill.sh [argocd_version_a] [argocd_version_b]  e.g. ./run-upgrade-drill.sh v2.14.11 v3.0.6
set -euo pipefail
VA="${1:-v2.14.11}"
VB="${2:-v3.0.6}"
CLUSTER="${CLUSTER_NAME:-argocd-lifecycle}"
NS=argocd

kind create cluster --name "$CLUSTER" --wait 60s || kind get clusters | grep -q "$CLUSTER"

install_bundle() { # $1 = version
  kubectl create namespace $NS --dry-run=client -o yaml | kubectl apply -f -
  kubectl apply -n $NS -f "https://raw.githubusercontent.com/argoproj/argo-cd/$1/manifests/install.yaml"
  kubectl -n $NS rollout status deploy/argocd-server --timeout=300s
}

echo "== install bundle-managed ArgoCD $VA =="
install_bundle "$VA"
kubectl -n $NS get deploy argocd-server -o jsonpath='{.spec.template.spec.containers[0].image}' > version-before.txt

echo "== seed a test Application =="
kubectl apply -f "$(dirname "$0")/test-application.yaml"
sleep 15
kubectl -n $NS get applications -o wide > apps-before-upgrade.txt

echo "== UPGRADE $VA -> $VB =="
install_bundle "$VB"
kubectl -n $NS get deploy argocd-server -o jsonpath='{.spec.template.spec.containers[0].image}' > version-after.txt
kubectl -n $NS get applications -o wide > apps-after-upgrade.txt

echo "== ROLLBACK $VB -> $VA =="
install_bundle "$VA"
kubectl -n $NS get applications -o wide > apps-after-rollback.txt

echo "Done. Compare version-*.txt and apps-*.txt in $(pwd)."
