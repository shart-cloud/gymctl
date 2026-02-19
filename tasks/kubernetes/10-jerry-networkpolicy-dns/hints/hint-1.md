Start with policy inspection:

```bash
kubectl get networkpolicy -o yaml
```

Then test DNS from the affected pod:

```bash
kubectl exec deploy/dns-debug -- nslookup kubernetes.default.svc.cluster.local
```
