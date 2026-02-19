Patch the host directly:

```bash
kubectl patch ingress jerry-web --type merge -p '{"spec":{"rules":[{"host":"jerry.local","http":{"paths":[{"path":"/","pathType":"Prefix","backend":{"service":{"name":"jerry-ingress-svc","port":{"number":80}}}}]}}]}}'
```
