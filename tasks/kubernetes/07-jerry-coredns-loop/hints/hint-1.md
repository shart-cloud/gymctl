Start with CoreDNS health and events:

```bash
kubectl get pods -n kube-system -l k8s-app=kube-dns
kubectl describe deployment coredns -n kube-system
kubectl get events -n kube-system --sort-by=.metadata.creationTimestamp | tail -40
```

Then confirm DNS is actually failing from a pod:

```bash
kubectl run dns-test --image=busybox:1.36 --restart=Never --command -- nslookup kubernetes.default.svc.cluster.local
kubectl logs dns-test
kubectl delete pod dns-test --ignore-not-found
```
