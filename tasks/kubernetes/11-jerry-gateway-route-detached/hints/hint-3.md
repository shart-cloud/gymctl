Patch the HTTPRoute parent reference:

```bash
kubectl patch httproute jerry-route --type merge -p '{"spec":{"parentRefs":[{"name":"shared-gateway","namespace":"gateway-shared"}]}}'
```

Re-check:

```bash
kubectl get httproute jerry-route -o jsonpath='{.spec.parentRefs[0].name}{"\n"}'
```
