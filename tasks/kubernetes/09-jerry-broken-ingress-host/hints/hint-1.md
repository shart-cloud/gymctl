Inspect the Ingress definition first:

```bash
kubectl get ingress jerry-web -o yaml
```

Focus on the `rules[].host` value.
