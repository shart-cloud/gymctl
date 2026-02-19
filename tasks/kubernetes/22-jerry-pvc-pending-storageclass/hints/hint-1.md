Check the PVC status and events first:

```bash
kubectl get pvc jerry-data
kubectl describe pvc jerry-data
kubectl get events --sort-by=.metadata.creationTimestamp | tail -30
```

Look for a `StorageClass` or provisioning-related error.
