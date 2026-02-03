# Hint 3: Complete solution

Edit the frontend deployment and fix the BACKEND_URL:

```yaml
env:
- name: BACKEND_URL
  value: "http://backend-api.backend.svc.cluster.local:8080"
```

Apply the changes:
```bash
kubectl apply -f frontend-deployment.yaml
```

Verify it works:
```bash
kubectl logs -l app=frontend-app
```