Inspect the CoreDNS ConfigMap:

```bash
kubectl -n kube-system get configmap coredns -o yaml
```

Look for a bad upstream forwarder (`forward . 127.0.0.1`) and a missing
`kubernetes cluster.local` plugin block in `Corefile`.
