# CKS-003 — Hint 1 (free)

Get the lay of the land:

```bash
kubectl -n kubernetes-dashboard get ingress dashboard -o yaml   # no tls: block
kubectl -n kubernetes-dashboard get ingress dashboard -o jsonpath='{.spec.tls}'  # empty
kubectl get clusterrolebinding | grep dashboard                 # bound to cluster-admin
kubectl auth can-i '*' '*' --all-namespaces \
  --as=system:serviceaccount:kubernetes-dashboard:dashboard      # yes  ← that's the problem
```

Two separate problems, two separate fixes:

1. **TLS** — the Ingress serves `dashboard.lab.local` over plain HTTP. It needs
   a `dashboard-tls` Secret and a `tls:` block, plus a redirect so HTTP can't be
   used at all.
2. **RBAC** — the dashboard's ServiceAccount is `cluster-admin`. If the pod is
   compromised, the whole cluster is. It should be able to *read* what a
   dashboard shows and nothing more.

The gym's kind cluster has no host port mappings, so test the live Ingress from
*inside* the cluster: curl the ingress-nginx controller's ClusterIP with
`--resolve dashboard.lab.local:443:<clusterIP>`.

Docs: <https://kubernetes.io/docs/concepts/services-networking/ingress/#tls>
