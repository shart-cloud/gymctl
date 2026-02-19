#!/bin/bash
set -e

# Check if metrics-server deployment exists and is available
kubectl get deployment metrics-server -n kube-system >/dev/null 2>&1

# Check if metrics-server has at least 1 ready replica
READY_REPLICAS=$(kubectl get deployment metrics-server -n kube-system -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")

if [[ "$READY_REPLICAS" -ge 1 ]]; then
    echo "✓ metrics-server is running with $READY_REPLICAS ready replicas"
    exit 0
else
    echo "✗ metrics-server is not ready (ready replicas: $READY_REPLICAS)"
    exit 1
fi