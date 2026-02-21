Start by understanding the current state:

```bash
kubectl get nodes
kubectl get pvc jerry-local-claim
kubectl get pv jerry-local-pv -o wide
kubectl describe pod jerry-local-app | tail -30
```

The pod is Pending. The Events section in `describe pod` will show you exactly why the
scheduler can't place it.

Then look at the PV's nodeAffinity:

```bash
kubectl get pv jerry-local-pv -o yaml | grep -A15 nodeAffinity
```

Which node does the PV require? What's the scheduling status of that node?

```bash
kubectl get nodes -o wide
kubectl describe node <the-required-node> | grep -E "Taints:|Unschedulable"
```
