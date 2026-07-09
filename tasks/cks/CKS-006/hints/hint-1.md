# CKS-006 — Hint 1 (free)

Look at what's mounted today:

```bash
kubectl -n production exec deploy/web-app -- ls /var/run/secrets/kubernetes.io/serviceaccount/
kubectl -n production get sa default -o yaml | grep -i automount
```

There are two levers:
- **ServiceAccount** level: `automountServiceAccountToken: false` on the SA.
- **Pod** level: `automountServiceAccountToken: false` in `spec` — this wins and
  is what removes the token file from a specific workload.

Only `api-server` needs the API, so only it should keep a token — as its own SA.

Docs: <https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/>
