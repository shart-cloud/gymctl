# CKS-005 — Hint 3 (cost 50)

```bash
kubectl delete clusterrolebinding ci-bot-cluster-admin
```

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata: { name: ci-bot-dev, namespace: dev-team }
rules:
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get","list","watch","create","update","patch","delete"]
  - apiGroups: [""]
    resources: ["services","configmaps"]
    verbs: ["get","list","watch","create","update","patch","delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata: { name: ci-bot-dev, namespace: dev-team }
subjects: [ { kind: ServiceAccount, name: ci-bot, namespace: dev-team } ]
roleRef: { kind: Role, name: ci-bot-dev, apiGroup: rbac.authorization.k8s.io }
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata: { name: ci-bot-readonly }
rules:
  - apiGroups: [""]
    resources: ["namespaces","nodes"]
    verbs: ["get","list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata: { name: ci-bot-readonly }
subjects: [ { kind: ServiceAccount, name: ci-bot, namespace: dev-team } ]
roleRef: { kind: ClusterRole, name: ci-bot-readonly, apiGroup: rbac.authorization.k8s.io }
```

Full manifests in `solution/`.
