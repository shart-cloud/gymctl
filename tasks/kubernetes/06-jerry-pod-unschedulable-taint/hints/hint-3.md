Remove the bad taint from all nodes, then verify rollout health:

```bash
kubectl taint nodes --all jerry-
kubectl rollout status deployment/jerry-worker
kubectl get pods -l app=jerry-worker
```

