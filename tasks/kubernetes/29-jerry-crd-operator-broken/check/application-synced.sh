#!/bin/bash
set -e

# Check if the Application exists
if ! kubectl get application jerry-simple-app -n argocd >/dev/null 2>&1; then
    echo "✗ Application jerry-simple-app not found"
    exit 1
fi

# Check the sync status
SYNC_STATUS=$(kubectl get application jerry-simple-app -n argocd -o jsonpath='{.status.sync.status}' 2>/dev/null || echo "Unknown")

if [[ "$SYNC_STATUS" == "Synced" ]]; then
    echo "✓ Application is Synced"
    exit 0
else
    echo "✗ Application sync status is '$SYNC_STATUS', expected 'Synced'"

    # Show more details for debugging
    echo "Application status:"
    kubectl get application jerry-simple-app -n argocd -o wide 2>/dev/null || true
    exit 1
fi