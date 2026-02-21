Patch the StatefulSet and force a pod recreation:

```bash
kubectl patch statefulset jerry-db --type='json' \
  -p='[{"op":"replace","path":"/spec/template/spec/containers/0/volumeMounts/0/mountPath","value":"/data"}]'

kubectl delete pod jerry-db-0
kubectl wait --for=condition=Ready pod/jerry-db-0 --timeout=120s
```

Verify the mount is correct:

```bash
kubectl exec jerry-db-0 -- df -h /data
kubectl exec jerry-db-0 -- cat /data/startup.log
```

Then prove data persists across a restart:

```bash
kubectl exec jerry-db-0 -- sh -c 'echo persist-check > /data/probe'
kubectl delete pod jerry-db-0
kubectl wait --for=condition=Ready pod/jerry-db-0 --timeout=120s
kubectl exec jerry-db-0 -- cat /data/probe
```

The file should still be there after the restart.
