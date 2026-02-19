Start by inspecting the route and gateway objects:

```bash
kubectl get gateway -n gateway-shared
kubectl get httproute jerry-route -o yaml
```

Focus on `spec.parentRefs`.
