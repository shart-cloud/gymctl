The PVC is Bound and the pod is Running — but that just means storage is allocated.
Check where the volume is actually mounted inside the container:

```bash
kubectl describe pod jerry-db-0 | grep -A10 "Mounts:"
kubectl get pod jerry-db-0 -o jsonpath='{.spec.containers[0].volumeMounts}' | python3 -m json.tool
```

Then check where data is actually going:

```bash
kubectl exec jerry-db-0 -- ls /data 2>/dev/null || echo "nothing at /data"
kubectl exec jerry-db-0 -- ls /cache 2>/dev/null || echo "nothing at /cache"
```

The startup script writes to `/data/startup.log`. Is that file where you expect it?
