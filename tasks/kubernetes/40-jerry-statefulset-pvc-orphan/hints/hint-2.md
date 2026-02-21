StatefulSet PVC names follow a deterministic pattern:
`<volumeClaimTemplate.name>-<statefulset.name>-<ordinal>`

So `data-jerry-db-0` belongs to a StatefulSet named `jerry-db` using `volumeClaimTemplate.name: data`.
Redeploying `jerry-db` with the same spec will automatically reconnect to these PVCs.

First, clean up the wrong deployment:

```bash
kubectl delete statefulset jerry-db-v2
kubectl delete pvc data-jerry-db-v2-0 data-jerry-db-v2-1
```

Then redeploy the original StatefulSet with the correct name (`jerry-db`).
