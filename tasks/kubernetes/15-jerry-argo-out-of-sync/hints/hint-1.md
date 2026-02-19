Inspect the Application first:

```bash
kubectl get applications.argoproj.io jerry-portfolio -n argocd -o yaml
```

Look closely at `spec.source.path` and `spec.destination.namespace`.
