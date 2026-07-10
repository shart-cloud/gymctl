# Hint 1 — Orientation (free)

Two Deployments run with **no** syscall confinement:

- `public/web-frontend`
- `monitoring/log-writer`

Look at their pod spec:

```bash
kubectl -n public     get deploy web-frontend -o jsonpath='{.spec.template.spec.securityContext.seccompProfile}{"\n"}'
kubectl -n monitoring get deploy log-writer   -o jsonpath='{.spec.template.spec.securityContext.seccompProfile}{"\n"}'
```

Both say `{"type":"Unconfined"}` — the kernel applies no seccomp filter, so
every syscall is allowed. That is the vulnerability.

The one idea: set each pod's `securityContext.seccompProfile` to a real profile.
There are two kinds — a runtime-provided one and a file-on-the-node one. One of
these workloads wants each. Both must stay **Running** after you change them.
