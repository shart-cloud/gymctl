`allowVolumeExpansion` is a field on the StorageClass. Unlike most StorageClass fields,
it CAN be patched on an existing class without recreating it.

Enable it:

```bash
kubectl patch storageclass expandable-local --type=merge \
  -p '{"allowVolumeExpansion": true}'

kubectl get storageclass expandable-local -o jsonpath='{.allowVolumeExpansion}'; echo
```

Then retry the PVC resize:

```bash
kubectl patch pvc jerry-app-data --type=merge \
  -p '{"spec":{"resources":{"requests":{"storage":"5Gi"}}}}'

kubectl get pvc jerry-app-data
```
