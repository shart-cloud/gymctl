# Hint 3: Restoring CNI Functionality

If the kindnet pod is missing or crashed, you need to restore it:

```bash
# Check DaemonSet status
kubectl get daemonset -n kube-system kindnet

# If pods are missing, try deleting and recreating them
kubectl delete pod -n kube-system -l app=kindnet

# The DaemonSet should automatically recreate the pods
kubectl get pods -n kube-system -l app=kindnet
```

Wait for all kindnet pods to be Running, then test pod creation:

```bash
kubectl run test-networking --image=nginx:1.20 --rm -i --restart=Never -- echo "Network test"
```

If CNI configuration files are actually missing from `/etc/cni/net.d/`, you would need to restore them from a working node or reinstall the CNI plugin.