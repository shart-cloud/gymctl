# Hint 1: Check Why VPA Has No Recommendations

VPA needs to observe actual resource usage to generate recommendations. Look at what it needs:

```bash
kubectl describe vpa jerry-app-vpa
```

Check the **Status** section. If `Recommendation` is empty, VPA can't see metrics.

VPA depends on **metrics-server** to read pod CPU and memory usage. Check if it's running:

```bash
kubectl get pods -n kube-system | grep -E 'vpa|metrics'
kubectl top pods
```

What do you see?
