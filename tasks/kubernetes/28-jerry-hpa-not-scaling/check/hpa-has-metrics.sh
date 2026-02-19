#!/bin/bash
set -e

# Check if HPA exists
kubectl get hpa jerry-app-hpa >/dev/null 2>&1

# Get the current CPU utilization from HPA status
# This should be a number, not "<unknown>"
CURRENT_CPU=$(kubectl get hpa jerry-app-hpa -o jsonpath='{.status.currentMetrics[0].resource.current.averageUtilization}' 2>/dev/null || echo "")

if [[ -n "$CURRENT_CPU" && "$CURRENT_CPU" =~ ^[0-9]+$ ]]; then
    echo "✓ HPA shows numeric CPU utilization: ${CURRENT_CPU}%"
    exit 0
else
    # Also check the TARGETS column which might show <unknown>/50%
    TARGETS=$(kubectl get hpa jerry-app-hpa -o custom-columns=TARGETS:.spec.targetCPUUtilizationPercentage --no-headers 2>/dev/null || echo "")
    HPA_OUTPUT=$(kubectl get hpa jerry-app-hpa 2>/dev/null || echo "")

    if echo "$HPA_OUTPUT" | grep -q "<unknown>"; then
        echo "✗ HPA shows <unknown> metrics"
        echo "HPA output: $HPA_OUTPUT"
        exit 1
    else
        echo "✗ HPA current metrics unavailable or invalid: '$CURRENT_CPU'"
        echo "HPA output: $HPA_OUTPUT"
        exit 1
    fi
fi