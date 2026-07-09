# CKS-021 — Hint 2 (cost 25)

The scanners flag the same set of controls on both workloads. Edit
`manifests/web.yaml` and `manifests/api.yaml` so each container's
`securityContext` (and the pod spec) has:

- `runAsNonRoot: true` and a non-zero `runAsUser` (root is the #1 kubesec CRITICAL)
- `readOnlyRootFilesystem: true`
- `allowPrivilegeEscalation: false`, `privileged: false`
- `capabilities: { drop: [ALL] }`
- `resources.requests` **and** `resources.limits` for cpu + memory
- a pinned image tag (not `:latest`)

And on the pod spec itself:

- remove `hostNetwork`, `hostPID`, `hostIPC`
- remove the `hostPath` volume (and its mount)

Two traps:

- **Read-only rootfs breaks nginx.** With `readOnlyRootFilesystem: true`, nginx
  can't write its pid/temp/cache and won't start. Give it writable scratch space
  with `emptyDir` volumes mounted at `/tmp`, `/var/cache/nginx`, and `/var/run`.
  Use the `nginxinc/nginx-unprivileged` image so it listens on `:8080` as a
  non-root user instead of needing to bind `:80` as root.
- **`sleep` needs nothing writable**, so the api container takes
  `readOnlyRootFilesystem: true` with no emptyDir. `busybox:1.36` runs fine as
  uid 1000.

Then apply and watch them roll out:

```bash
kubectl apply -f manifests/
kubectl -n staging rollout status deploy/web
kubectl -n staging rollout status deploy/api
```
