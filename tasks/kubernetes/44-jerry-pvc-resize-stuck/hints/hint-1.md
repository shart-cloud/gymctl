Try to resize the PVC and observe the error:

```bash
kubectl patch pvc jerry-app-data --type=merge \
  -p '{"spec":{"resources":{"requests":{"storage":"5Gi"}}}}'
```

The API server should reject this with a message about volume expansion.

Now check why:

```bash
kubectl get storageclass expandable-local -o yaml | grep -i expansion
kubectl describe storageclass expandable-local
```

The `allowVolumeExpansion` field controls whether PVCs using this class can be resized.
