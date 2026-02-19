# Hint 2: HPA Needs metrics-server

HPA needs **metrics-server** to read CPU/memory utilization from pods.

Check if metrics-server is running:

```bash
kubectl get pods -n kube-system | grep metrics
```

If no metrics-server pods are running, the HPA can't get CPU metrics.

Also try:

```bash
kubectl top nodes
```

If this command fails or shows no data, metrics-server is missing or broken.