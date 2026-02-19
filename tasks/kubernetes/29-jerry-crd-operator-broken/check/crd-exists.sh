#!/bin/bash
set -e

# Check if the Application CRD exists
if kubectl get crd applications.argoproj.io >/dev/null 2>&1; then
    echo "✓ Application CRD exists"
    exit 0
else
    echo "✗ Application CRD not found"
    echo "Available CRDs:"
    kubectl get crd | grep -i application || echo "No application CRDs found"
    exit 1
fi