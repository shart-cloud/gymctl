#!/bin/bash
set -e

# Check if the ArgoCD application controller deployment exists and has replicas > 0
DEPLOYMENT_EXISTS=$(kubectl get deployment argocd-application-controller -n argocd >/dev/null 2>&1 && echo "true" || echo "false")

if [[ "$DEPLOYMENT_EXISTS" != "true" ]]; then
    echo "✗ ArgoCD application controller deployment not found"
    exit 1
fi

# Check replicas
DESIRED_REPLICAS=$(kubectl get deployment argocd-application-controller -n argocd -o jsonpath='{.spec.replicas}')
READY_REPLICAS=$(kubectl get deployment argocd-application-controller -n argocd -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")

if [[ "$DESIRED_REPLICAS" -gt 0 && "$READY_REPLICAS" -gt 0 ]]; then
    echo "✓ ArgoCD application controller is running ($READY_REPLICAS/$DESIRED_REPLICAS ready)"
    exit 0
else
    echo "✗ ArgoCD application controller not running (desired: $DESIRED_REPLICAS, ready: $READY_REPLICAS)"
    exit 1
fi