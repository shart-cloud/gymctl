# CKS-005 — Hint 1 (free)

See what ci-bot can do today, and find the grant:

```bash
S=system:serviceaccount:dev-team:ci-bot
kubectl auth can-i --list --as=$S | head
kubectl get clusterrolebindings -o wide | grep ci-bot
```

Least privilege has two moves: **remove** the broad grant, then **add back**
only the specific verbs/resources CI needs. Namespaced things → `Role`;
cluster-scoped things (nodes, namespaces) → `ClusterRole`.

Docs: <https://kubernetes.io/docs/reference/access-authn-authz/rbac/>
