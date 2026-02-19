#!/bin/bash
set -e

# Check if the Application has synced recently (within last 10 minutes)
LAST_SYNC=$(kubectl get application jerry-simple-app -n argocd -o jsonpath='{.status.operationState.finishedAt}' 2>/dev/null || echo "")

if [[ -z "$LAST_SYNC" ]]; then
    echo "✗ Application has no recent sync operation"
    exit 1
fi

# Convert timestamp to epoch seconds for comparison
# This is a simplified check - in a real scenario you'd want more robust date handling
CURRENT_TIME=$(date -u +%s)
SYNC_TIME=$(date -u -d "$LAST_SYNC" +%s 2>/dev/null || echo "0")

# Check if sync happened within last 10 minutes (600 seconds)
TIME_DIFF=$((CURRENT_TIME - SYNC_TIME))

if [[ "$TIME_DIFF" -lt 600 ]]; then
    echo "✓ Application synced recently (${TIME_DIFF}s ago)"
    exit 0
else
    echo "✗ Application last synced too long ago (${TIME_DIFF}s ago)"
    echo "Last sync time: $LAST_SYNC"
    exit 1
fi