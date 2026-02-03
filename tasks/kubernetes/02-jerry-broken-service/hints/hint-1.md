# Hint 1: Check the endpoints

Run these commands to debug:
```bash
kubectl get endpoints jerry-app-service
kubectl get pods --show-labels
kubectl describe svc jerry-app-service
```

Compare the Service selector with the Pod labels.