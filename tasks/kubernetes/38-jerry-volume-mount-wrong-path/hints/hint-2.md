The StatefulSet spec has `mountPath: /cache` but the container command writes to `/data/startup.log`.
Writes to `/data` land on ephemeral overlay storage, not the PVC.

You need to patch the StatefulSet to change `mountPath` from `/cache` to `/data`.

StatefulSet pod template changes are not applied to running pods automatically — delete
the pod after patching so the StatefulSet controller recreates it with the corrected spec:

```bash
kubectl edit statefulset jerry-db
# Change: mountPath: /cache  →  mountPath: /data
kubectl delete pod jerry-db-0
kubectl wait --for=condition=Ready pod/jerry-db-0 --timeout=120s
```
