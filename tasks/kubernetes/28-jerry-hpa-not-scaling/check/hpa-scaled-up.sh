#!/bin/bash
set -e

# Check current replicas of the deployment
CURRENT_REPLICAS=$(kubectl get deployment jerry-app -o jsonpath='{.status.replicas}' 2>/dev/null || echo "0")

# Check HPA min replicas (should be 1)
MIN_REPLICAS=$(kubectl get hpa jerry-app-hpa -o jsonpath='{.spec.minReplicas}' 2>/dev/null || echo "1")

if [[ "$CURRENT_REPLICAS" -gt "$MIN_REPLICAS" ]]; then
    echo "✓ HPA has scaled up: $CURRENT_REPLICAS replicas (min: $MIN_REPLICAS)"
    exit 0
else
    echo "✗ HPA has not scaled up: $CURRENT_REPLICAS replicas (min: $MIN_REPLICAS)"

    # Show HPA status for debugging
    echo "HPA status:"
    kubectl describe hpa jerry-app-hpa | tail -10 || true
    exit 1
fi