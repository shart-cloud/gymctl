Start with cluster symptoms:

```bash
kubectl get nodes
kubectl describe node $(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')
kubectl get events --all-namespaces --sort-by=.metadata.creationTimestamp | tail -30
```

Look for node health and heartbeat-related signals.
