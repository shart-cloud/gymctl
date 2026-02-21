The PVC requests `ReadWriteMany` but `rancher.io/local-path` only supports `ReadWriteOnce`.
Local storage is node-local — it can only be mounted by pods on a single node at a time.

`accessModes` is immutable on an existing PVC, so you need to delete and recreate it:

```bash
kubectl delete pod jerry-rwx-app --ignore-not-found
kubectl delete pvc jerry-rwx-claim
```

Then recreate the PVC with `ReadWriteOnce` and reapply the pod.
