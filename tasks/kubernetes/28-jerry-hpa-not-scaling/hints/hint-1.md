# Hint 1: Check HPA Status

The HPA exists but something is wrong with metrics collection.

Check the HPA status and conditions:

```bash
kubectl describe hpa jerry-app-hpa
```

Look at the **Conditions** section at the bottom. What does it say about metrics?

Also try:

```bash
kubectl get hpa jerry-app-hpa
```

What does the **TARGETS** column show? If you see `<unknown>/50%`, the HPA can't read CPU metrics.