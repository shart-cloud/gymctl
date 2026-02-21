# Hint 1: Check What Is Running in kube-system

Pods are stuck Pending. Start with the cluster's control plane:

```bash
kubectl get pods -n kube-system
kubectl get events --sort-by=.lastTimestamp | tail -20
```

Compare what you see against the expected control-plane components. Which
component is absent? What do the events say about why pods are not being
scheduled?
