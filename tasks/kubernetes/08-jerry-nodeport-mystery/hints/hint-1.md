Start by inspecting the Service and pod ports side-by-side:

```bash
kubectl get svc jerry-nodeport -o yaml
kubectl get deploy jerry-nodeport-app -o yaml
```
