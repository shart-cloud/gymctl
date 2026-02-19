# Hint 2: CNI System Pods

In kind clusters, networking is provided by kindnet. Check the CNI-related system pods:

```bash
kubectl get pods -n kube-system | grep kindnet
kubectl describe pod -n kube-system -l app=kindnet
```

If the kindnet DaemonSet pod is missing or failing on a node, new pods on that node can't get IP addresses.

The CNI configuration files are located at `/etc/cni/net.d/` on each node. You can inspect this inside kind nodes with `docker exec`.