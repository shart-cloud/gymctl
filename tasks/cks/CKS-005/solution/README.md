# CKS-005 — Solution

Least privilege is *remove the wildcard, then add back only what's needed*.

```bash
# 1. Revoke cluster-admin (apply cannot delete for you)
kubectl delete clusterrolebinding ci-bot-cluster-admin

# 2. Grant the scoped permissions
kubectl apply -f solution/rbac.yaml
```

## Why

- **Role/RoleBinding `ci-bot-dev`** — namespaced. CI only ever touches its own
  namespace, so deployments/services/configmaps are a *Role*, not a ClusterRole.
- **ClusterRole/ClusterRoleBinding `ci-bot-readonly`** — some reads (nodes,
  namespaces) are cluster-scoped, so they must be a ClusterRole. Verbs are
  `get`/`list` only — no writes, no wildcards.
- Secrets are never granted, so `can-i get secrets` is `no`.

## Verify

```bash
S=system:serviceaccount:dev-team:ci-bot
kubectl auth can-i create deployments -n dev-team --as=$S   # yes
kubectl auth can-i list nodes                 --as=$S       # yes
kubectl auth can-i delete pods   -n dev-team  --as=$S       # no
kubectl auth can-i get secrets   -n dev-team  --as=$S       # no
kubectl auth can-i '*' '*' --all-namespaces   --as=$S       # no
```
