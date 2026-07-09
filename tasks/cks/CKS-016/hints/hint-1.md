# CKS-016 — Hint 1 (free)

Pod Security Admission is built into the API server. You turn it on per
namespace with labels — no webhook, no install:

```
pod-security.kubernetes.io/<mode>: <level>
```

- modes: `enforce` (block), `warn`, `audit`
- levels: `privileged`, `baseline`, `restricted`

Test the effect without creating anything using server-side dry-run:

```bash
kubectl -n restricted-apps run probe --image=busybox:1.36 --dry-run=server -- sleep 1
```

Docs: <https://kubernetes.io/docs/tasks/configure-pod-container/enforce-standards-namespace-labels/>
