# Hint 3: kubeadm Can Regenerate the Manifest

You don't need a backup. `kubeadm` can regenerate any control-plane manifest:

```bash
node=$(kubectl get nodes -l node-role.kubernetes.io/control-plane \
  -o jsonpath='{.items[0].metadata.name}')
docker exec "$node" kubeadm init phase control-plane scheduler
```

After this, kubelet detects the new file and starts the scheduler pod
automatically. Watch recovery:

```bash
kubectl get pods -n kube-system -w
```
