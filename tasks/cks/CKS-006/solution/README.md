# CKS-006 — Solution

```bash
kubectl apply -f solution/hardened.yaml
```

## Why each piece

- **`default` SA `automountServiceAccountToken: false`** — defence in depth: any
  pod that forgets to opt out still gets no token.
- **`web-app` / `worker` pod specs `automountServiceAccountToken: false`** — the
  pod-level setting is what actually removes the mounted token for these
  workloads. The whole `/var/run/secrets/...serviceaccount` projection
  disappears, so there's no token file at all.
- **`api-sa` + Role/RoleBinding** — the only workload that needs the API gets
  its own SA scoped to `get`/`list` on secrets and configmaps.
- **`api-server` uses `api-sa`** and keeps `automountServiceAccountToken: true`,
  so its token is present.

## Verify

```bash
kubectl -n production exec deploy/web-app    -- ls /var/run/secrets/kubernetes.io/serviceaccount/ 2>&1 || echo "no token (good)"
kubectl -n production exec deploy/api-server -- cat /var/run/secrets/kubernetes.io/serviceaccount/token >/dev/null && echo "api-server has token (good)"
kubectl -n production get sa default -o jsonpath='{.automountServiceAccountToken}{"\n"}'   # false
```
