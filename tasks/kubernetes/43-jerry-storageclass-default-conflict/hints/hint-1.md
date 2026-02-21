Check the StorageClass list:

```bash
kubectl get storageclass
```

Look for `(default)` appearing next to more than one class — that's the problem.
Inspect both:

```bash
kubectl describe storageclass standard-local | grep -i default
kubectl describe storageclass fast-local | grep -i default
```

Also check the PVC status:

```bash
kubectl describe pvc jerry-unqualified-pvc
kubectl get events --sort-by=.metadata.creationTimestamp | tail -20
```
