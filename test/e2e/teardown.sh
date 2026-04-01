#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="${KIND_CLUSTER:-supabase-operator-test-e2e}"
echo "=== Deleting kind cluster: $CLUSTER_NAME ==="
kind delete cluster --name "$CLUSTER_NAME"
