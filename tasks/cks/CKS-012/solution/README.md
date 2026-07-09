# CKS-012 — Solution

```bash
kubectl apply -f solution/backend-api.yaml
```

## The hardened baseline

| Setting | Where | Why |
| --- | --- | --- |
| `runAsUser/runAsGroup: 1000`, `runAsNonRoot: true` | pod `securityContext` | never run as root |
| `readOnlyRootFilesystem: true` | container `securityContext` | attacker can't modify the image at runtime |
| `allowPrivilegeEscalation: false` | container `securityContext` | block setuid/`no_new_privs` bypass |
| `capabilities.drop: ["ALL"]` + `add: ["NET_BIND_SERVICE"]` | container `securityContext` | start from zero, add only what's needed |
| `emptyDir` at `/tmp` | volume + volumeMount | the app still needs scratch space under a read-only root |

## The catch

A read-only root filesystem breaks anything that writes to disk. The app needs
somewhere to write, so mount an `emptyDir` at `/tmp` (or wherever it writes).
Without it, the container may crash-loop and the "Available" check fails.

## Verify

```bash
kubectl -n app exec deploy/backend-api -- id -u                       # 1000
kubectl -n app exec deploy/backend-api -- sh -c 'echo ok > /tmp/x'    # succeeds
kubectl -n app exec deploy/backend-api -- sh -c 'echo x > /etc/x'     # fails (read-only)
```
