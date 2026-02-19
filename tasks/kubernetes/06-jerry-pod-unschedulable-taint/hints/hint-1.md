Pods in `Pending` usually means the scheduler cannot place them.

Start with:

```bash
kubectl get pods
kubectl describe pod <pod-name>
kubectl get events --sort-by=.metadata.creationTimestamp | tail -30
```

