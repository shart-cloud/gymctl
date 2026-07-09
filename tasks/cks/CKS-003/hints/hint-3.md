# CKS-003 — Hint 3 (cost 50)

Patched Ingress (tls block + redirect):

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: dashboard
  namespace: kubernetes-dashboard
  annotations:
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
    - hosts: ["dashboard.lab.local"]
      secretName: dashboard-tls
  rules:
    - host: dashboard.lab.local
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: dashboard
                port: { number: 80 }
```

Least-privilege RBAC (replace cluster-admin):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata: { name: dashboard-viewer }
rules:
  - apiGroups: [""]
    resources: ["pods","services","namespaces","nodes","configmaps","endpoints","events","persistentvolumeclaims"]
    verbs: ["get","list","watch"]
  - apiGroups: ["apps"]
    resources: ["deployments","replicasets","statefulsets","daemonsets"]
    verbs: ["get","list","watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata: { name: dashboard-viewer }
subjects:
  - kind: ServiceAccount
    name: dashboard
    namespace: kubernetes-dashboard
roleRef:
  kind: ClusterRole
  name: dashboard-viewer
  apiGroup: rbac.authorization.k8s.io
```

Then the one step `apply` can't do for you:

```bash
kubectl delete clusterrolebinding dashboard-cluster-admin
```

Full manifests (including a ready-made `dashboard-tls` Secret) are in
`solution/`. `kubectl apply -f solution/` + that one delete drives every check
green.
