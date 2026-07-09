# CKS-024 — Hint 1 (free)

Confirm the containers are mutable right now:

```bash
kubectl -n production exec deploy/web -- sh -c 'echo x > /etc/passwd.bak && echo WRITABLE'
```

The fix is the same three settings on every deployment's **container**
`securityContext`:

- `readOnlyRootFilesystem: true`
- `allowPrivilegeEscalation: false`
- plus an `emptyDir` volume mounted at `/tmp` (a read-only root breaks anything
  that writes to disk — give it scratch space back).

Docs: <https://kubernetes.io/docs/tasks/configure-pod-container/security-context/>
