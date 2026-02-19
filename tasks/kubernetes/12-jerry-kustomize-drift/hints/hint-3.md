Patch prod back to base-compatible values:

```bash
kubectl -n kustomize-prod patch deployment course-web --type merge -p '{"spec":{"selector":{"matchLabels":{"app":"course-web"}},"template":{"metadata":{"labels":{"app":"course-web"}},"spec":{"containers":[{"name":"web","image":"nginx:1.27","env":[{"name":"APP_MODE","value":"prod"}]}]}}}}'
```

Then verify service endpoints:

```bash
kubectl get endpoints course-web -n kustomize-prod
```
