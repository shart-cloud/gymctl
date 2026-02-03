# Hint 1: Observe the problem

Watch the pods restart:
```bash
kubectl get pods -w
kubectl describe pod <pod-name>
```

Look for:
- Restart count
- Liveness probe failed events
- How long before the first restart?