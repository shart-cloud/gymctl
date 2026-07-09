# CKS-021 — Solution

Scan, fix, re-scan, apply. The hardened manifests live here; apply them to
`staging`:

```bash
kubectl apply -f solution/
kubectl -n staging rollout status deploy/web
kubectl -n staging rollout status deploy/api
```

## The workflow (shift-left)

```bash
# 1. scan the broken manifests and save the reports
mkdir -p reports
docker run --rm -i kubesec/kubesec:v2 scan /dev/stdin < manifests/web.yaml > reports/kubesec.json
docker run --rm -v "$PWD/manifests:/m" stackrox/kube-linter:v0.7.4 lint /m   > reports/kube-linter.txt 2>&1

# 2. fix every CRITICAL/HIGH in the manifest files (see table below)
# 3. re-scan to confirm the CRITICALs are gone
# 4. apply and verify the workloads run
kubectl apply -f manifests/
```

## Findings → fixes, and WHY each fix matters

| Scanner finding | Field changed | Why it matters |
| --- | --- | --- |
| kubesec CRITICAL `Privileged` / kube-linter `privileged-container` | `securityContext.privileged: false` | A privileged container gets almost unrestricted host access — it can load kernel modules, access all devices, and escape to the node. |
| kubesec CRITICAL runs as root / kube-linter `run-as-non-root` | `runAsNonRoot: true`, `runAsUser: 101`/`1000` | Root in the container is root on the host if anything escapes. Non-root shrinks the blast radius of every other vuln. |
| kube-linter `privilege-escalation-container` | `allowPrivilegeEscalation: false` | Blocks `setuid`/`setgid` binaries from raising privileges past the container's own user (no `no_new_privs` bypass). |
| kubesec advises capabilities drop / kube-linter caps | `capabilities.drop: [ALL]` | Containers get a default capability set (e.g. `NET_RAW`, `CHOWN`). Dropping ALL removes every ambient kernel privilege; add back only what you truly need. |
| kube-linter `no-read-only-root-fs` | `readOnlyRootFilesystem: true` (+ `emptyDir` scratch) | A writable rootfs lets an attacker drop tools, tamper with binaries, or persist. Read-only forces all writes into declared, ephemeral volumes. |
| kube-linter `unset-cpu-requirements` / `unset-memory-requirements` | `resources.requests` + `resources.limits` | No limits means one workload can starve the node (a noisy-neighbour / DoS risk). Limits bound the damage. |
| host access (broken `api`) | removed `hostNetwork`, `hostPID`, and the `hostPath: /` volume | Host namespaces and a hostPath mount of `/` erase the container boundary — the pod can see host processes, host network, and every file on the node. |
| kube-linter `latest-tag` | pinned `nginxinc/nginx-unprivileged:1.27-alpine`, `busybox:1.36` | `:latest` is mutable — the image you scanned may not be the image that runs. Pin the tag so the scan result is meaningful. |

## The read-only-rootfs trap

`readOnlyRootFilesystem: true` stops the container writing anywhere on its own
filesystem. A `sleep` (the `api` container) needs nothing writable, so it just
works. **nginx does not** — it writes a pid file, temp files, and a proxy cache.
The fix is not to loosen the rootfs; it's to give nginx small writable
`emptyDir` volumes exactly where it needs them (`/tmp`, `/var/cache/nginx`,
`/var/run`) and use the `nginx-unprivileged` image so it binds `:8080` as a
non-root user instead of needing root to bind `:80`.

## Verify by hand

```bash
kubectl -n staging get deploy web api
kubectl -n staging get pods -o wide
# controls, straight off the applied objects:
kubectl -n staging get deploy web -o jsonpath='{.spec.template.spec.containers[0].securityContext}'; echo
kubectl -n staging get deploy api -o jsonpath='{.spec.template.spec.containers[0].securityContext}'; echo
```
