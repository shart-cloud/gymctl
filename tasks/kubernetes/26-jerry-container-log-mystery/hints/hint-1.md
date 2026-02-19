Start by identifying which container is failing:

```bash
kubectl get pods -l app=jerry-log-mystery
kubectl describe pod -l app=jerry-log-mystery
```

Check container status and restart counts, not just pod phase.
