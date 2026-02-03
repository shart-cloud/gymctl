# Hint 1: Check pod status

Run these commands to see the problem:
```bash
kubectl get pods
kubectl describe pod <pod-name>
```

Look for the error message about the missing ConfigMap.