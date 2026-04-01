#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="${KIND_CLUSTER:-supabase-operator-test-e2e}"
CNPG_VERSION="1.25.1"
GATEWAY_API_VERSION="1.2.1"
ENVOY_GATEWAY_VERSION="1.3.0"

echo "=== Creating kind cluster: $CLUSTER_NAME ==="
kind create cluster --config test/kind-config.yaml --name "$CLUSTER_NAME" --wait 60s

echo "=== Installing CNPG operator v$CNPG_VERSION ==="
kubectl apply --server-side -f "https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.25/releases/cnpg-${CNPG_VERSION}.yaml"
kubectl wait --for=condition=Available deployment/cnpg-controller-manager -n cnpg-system --timeout=120s
echo "CNPG operator ready"

echo "=== Installing Gateway API CRDs v$GATEWAY_API_VERSION ==="
kubectl apply -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/v${GATEWAY_API_VERSION}/standard-install.yaml"

echo "=== Installing Envoy Gateway v$ENVOY_GATEWAY_VERSION ==="
kubectl apply --server-side -f "https://github.com/envoyproxy/gateway/releases/download/v${ENVOY_GATEWAY_VERSION}/install.yaml"
kubectl wait --for=condition=Available deployment/envoy-gateway -n envoy-gateway-system --timeout=120s
echo "Envoy Gateway ready"

echo "=== Creating GatewayClass ==="
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: envoy-gateway
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
EOF

echo "=== Creating supabase-system namespace ==="
kubectl create namespace supabase-system --dry-run=client -o yaml | kubectl apply -f -

echo "=== Kind cluster ready for E2E tests ==="
