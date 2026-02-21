The pod is stuck in ContainerCreating or Error. Check what's happening:

```bash
kubectl describe pod jerry-projected-app | grep -A20 "Events:"
kubectl get pod jerry-projected-app
```

Then look at the projected volume spec:

```bash
kubectl get pod jerry-projected-app -o jsonpath='{.spec.volumes[0]}' | python3 -m json.tool
```

Compare this to what `kubectl explain` says the correct field names are:

```bash
kubectl explain pod.spec.volumes.projected.sources
kubectl explain pod.spec.volumes.projected.sources.configMap
kubectl explain pod.spec.volumes.projected.sources.secret
```
