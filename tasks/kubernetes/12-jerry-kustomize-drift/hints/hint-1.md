Compare dev and prod deployments directly:

```bash
kubectl get deploy course-web -n kustomize-dev -o yaml
kubectl get deploy course-web -n kustomize-prod -o yaml
```
