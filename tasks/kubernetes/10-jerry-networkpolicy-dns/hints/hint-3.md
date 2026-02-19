Patch the DNS allow policy to permit UDP 53 to CoreDNS in `kube-system`:

```bash
kubectl patch networkpolicy allow-dns-egress --type merge -p '{"spec":{"egress":[{"to":[{"namespaceSelector":{"matchLabels":{"kubernetes.io/metadata.name":"kube-system"}},"podSelector":{"matchLabels":{"k8s-app":"kube-dns"}}}],"ports":[{"protocol":"UDP","port":53}]}]}}'
```

Retest with:

```bash
kubectl exec deploy/dns-debug -- nslookup kubernetes.default.svc.cluster.local
```
