`Delete` removes PVs when PVCs are deleted.
If this class should preserve data for investigation/recovery, use `Retain`.

Check whether editing is accepted:

```bash
kubectl edit storageclass jerry-retain
```
