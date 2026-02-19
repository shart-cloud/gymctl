Check node taints directly:

```bash
kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints
```

Look for a taint that would block normal workloads (`NoSchedule`).

