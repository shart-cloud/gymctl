After context recovery, confirm target access and namespace behavior:

```bash
kubectl config set-context --current --namespace=default
kubectl get deploy jerry-context-app
kubectl get pods -l app=jerry-context-app
```

If these commands work, your context targeting is fixed.
