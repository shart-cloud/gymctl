Start by understanding what's currently in the cluster:

```bash
kubectl get statefulsets
kubectl get pvc
kubectl get pods
```

You'll see PVCs named `data-jerry-db-0` and `data-jerry-db-1` — those are from the
original StatefulSet. There's also a running `jerry-db-v2` with its own PVCs.

The original data is in `data-jerry-db-0` and `data-jerry-db-1`. Check their status:

```bash
kubectl describe pvc data-jerry-db-0
kubectl describe pvc data-jerry-db-1
```

What's the phase? Who last owned them?
