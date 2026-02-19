Start with control-plane symptoms:

```bash
kubectl -n kube-system get pods -l component=kube-scheduler
kubectl -n kube-system describe pod -l component=kube-scheduler
kubectl get pods -A | grep Pending
```

If kube-scheduler is unhealthy, new pods usually stay Pending.
