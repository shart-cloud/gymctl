# Hint 1: Check the namespaces

List resources in different namespaces:
```bash
kubectl get namespaces
kubectl get all -n backend
kubectl get all -n default
```

Try to reach the backend from the default namespace:
```bash
kubectl run test --rm -it --image=busybox -- wget -O- http://backend-api:8080/health
```

Why does it fail?