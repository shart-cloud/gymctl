Inspect the probe configuration on the pod template:

```bash
kubectl get deploy jerry-rollout-app -o yaml | grep -A8 readinessProbe:
kubectl describe pod -l app=jerry-rollout-app
```

The app serves standard nginx content on `/`.
