Start with rollout and availability signals:

```bash
kubectl get deploy jerry-rollout-app
kubectl rollout status deployment/jerry-rollout-app
kubectl describe deploy jerry-rollout-app
```

Check why pods are not becoming Ready.
