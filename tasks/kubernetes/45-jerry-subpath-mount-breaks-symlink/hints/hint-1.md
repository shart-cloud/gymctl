First, confirm the ConfigMap has the new value and the container has the old one:

```bash
kubectl get configmap jerry-app-config -o jsonpath='{.data.db-host}'; echo
kubectl exec jerry-config-app -- cat /etc/app/db-host
```

They're different. The ConfigMap updated but the container didn't.

Look at how the volume is mounted:

```bash
kubectl get pod jerry-config-app -o jsonpath='{.spec.containers[0].volumeMounts}' | python3 -m json.tool
```

Notice the `subPath` field. That's the clue.
