Restore a valid Corefile (with `kubernetes cluster.local`) and remove localhost
forwarding, then restart CoreDNS:

```bash
kubectl -n kube-system patch configmap coredns --type merge -p '{"data":{"Corefile":".:53 {\n    errors\n    health\n    ready\n    kubernetes cluster.local in-addr.arpa ip6.arpa {\n      pods insecure\n      fallthrough in-addr.arpa ip6.arpa\n      ttl 30\n    }\n    prometheus :9153\n    forward . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n"}}'
kubectl -n kube-system rollout restart deployment/coredns
kubectl -n kube-system rollout status deployment/coredns
```

Then validate DNS from a temporary pod:

```bash
kubectl run dns-test --image=busybox:1.36 --restart=Never --command -- nslookup kubernetes.default.svc.cluster.local
kubectl logs dns-test
kubectl delete pod dns-test
```
