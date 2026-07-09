# CKS-006 — Hint 3 (cost 50)

```yaml
apiVersion: v1
kind: ServiceAccount
metadata: { name: default, namespace: production }
automountServiceAccountToken: false
---
apiVersion: v1
kind: ServiceAccount
metadata: { name: api-sa, namespace: production }
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata: { name: api-sa, namespace: production }
rules:
  - apiGroups: [""]
    resources: ["secrets","configmaps"]
    verbs: ["get","list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata: { name: api-sa, namespace: production }
subjects: [ { kind: ServiceAccount, name: api-sa, namespace: production } ]
roleRef: { kind: Role, name: api-sa, apiGroup: rbac.authorization.k8s.io }
```

Then, in the `web-app` and `worker` deployments, under `spec.template.spec`:

```yaml
      automountServiceAccountToken: false
```

and in `api-server` under `spec.template.spec`:

```yaml
      serviceAccountName: api-sa
```

Full manifests in `solution/hardened.yaml`.
