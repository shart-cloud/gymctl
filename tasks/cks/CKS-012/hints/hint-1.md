# CKS-012 — Hint 1 (free)

See how exposed it is now:

```bash
kubectl -n app exec deploy/backend-api -- id           # uid=0(root)
kubectl -n app exec deploy/backend-api -- sh -c 'echo x > /etc/x && echo WRITABLE'
```

There are two `securityContext` blocks:
- **pod** `spec.template.spec.securityContext` — identity: `runAsUser`,
  `runAsGroup`, `runAsNonRoot`.
- **container** `...containers[0].securityContext` — capabilities,
  `readOnlyRootFilesystem`, `allowPrivilegeEscalation`.

The one thing that will break the app: a read-only root filesystem. Give it a
writable `emptyDir` at `/tmp`.

Docs: <https://kubernetes.io/docs/tasks/configure-pod-container/security-context/>
