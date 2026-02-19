Patch the readiness path and watch rollout recover:

```bash
kubectl patch deployment jerry-rollout-app \
  --type='json' \
  -p='[{"op":"replace","path":"/spec/template/spec/containers/0/readinessProbe/httpGet/path","value":"/"}]'

kubectl rollout status deployment/jerry-rollout-app
kubectl get deploy jerry-rollout-app
```
