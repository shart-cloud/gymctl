Compare the claim's class to available classes:

```bash
kubectl get storageclass
kubectl get pvc jerry-data -o jsonpath='{.spec.storageClassName}'; echo
```

Either create the missing class or update the claim to use an existing one.
