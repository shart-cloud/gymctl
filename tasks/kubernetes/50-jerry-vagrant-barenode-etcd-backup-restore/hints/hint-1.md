Start with node access and bootstrap if needed:

```bash
gymctl ssh cp-1
gymctl ssh worker-1
```

From cp-1, verify cluster readiness before etcd backup work:

```bash
export KUBECONFIG=/etc/kubernetes/admin.conf
kubectl get nodes
```
