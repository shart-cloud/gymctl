# CKS-016 — Hint 3 (cost 50)

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: restricted-apps
  labels:
    pod-security.kubernetes.io/enforce: restricted
---
apiVersion: v1
kind: Namespace
metadata:
  name: baseline-apps
  labels:
    pod-security.kubernetes.io/enforce: baseline
    pod-security.kubernetes.io/warn: restricted
```

```bash
kubectl apply -f solution/namespaces-labeled.yaml
```

A `restricted`-compliant pod needs `runAsNonRoot: true`, a seccomp profile
(`RuntimeDefault`), `allowPrivilegeEscalation: false`, and `capabilities.drop:
["ALL"]`. Full manifest in `solution/`.
