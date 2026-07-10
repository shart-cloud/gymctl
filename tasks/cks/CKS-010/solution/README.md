# CKS-010 — Solution

Two Deployments start with `seccompProfile.type: Unconfined` — every syscall
allowed. The fix moves each onto a real profile, applied through the pod-level
`securityContext.seccompProfile`.

```bash
kubectl apply -f solution/
# -> deployment web-frontend  (public)     RuntimeDefault
# -> deployment log-writer    (monitoring) Localhost profiles/audit-write-only.json
```

`kubectl apply -f solution/` is the whole fix. The Deployments keep their
original name and namespace, so `apply` updates them in place and the pods roll
onto the new profile. It is idempotent — apply it as many times as you like.

## Why each piece

| Workload | Profile | What it means |
| --- | --- | --- |
| `web-frontend` (public) | `type: RuntimeDefault` | Use the container runtime's built-in default seccomp profile. Ships with runc/containerd — **no node file needed**. Blocks a curated set of dangerous syscalls (including `unshare(CLONE_NEWUSER)`) while allowing what normal apps need. |
| `log-writer` (monitoring) | `type: Localhost` + `localhostProfile: profiles/audit-write-only.json` | Use a custom profile **file on the node**. The path is relative to the kubelet seccomp root `/var/lib/kubelet/seccomp`, so it resolves to `/var/lib/kubelet/seccomp/profiles/audit-write-only.json`. |

## RuntimeDefault vs Localhost — the distinction

- **RuntimeDefault** is the runtime's baked-in profile. It is the right default
  for almost everything: zero node-side setup, and it already blocks the scary
  syscalls. Prefer it unless you have a specific reason not to.
- **Localhost** points at a *file you control* on the node. Use it when you need
  a custom allow/deny/audit policy that differs from the runtime default. The
  cost is operational: the file must exist on **every** node a pod might land on,
  under `/var/lib/kubelet/seccomp/`, before the pod starts.

## Where the Localhost profile lives (and who put it there)

The node-side file `/var/lib/kubelet/seccomp/profiles/audit-write-only.json` is
provided by the **exercise setup**, not by you — placing files on nodes is an
operator task, not something a manifest can do. Its contents:

```json
{"defaultAction":"SCMP_ACT_LOG"}
```

`SCMP_ACT_LOG` is an **audit** action: it *allows* every syscall but logs it to
the kernel audit log. So `log-writer` keeps running while every syscall it makes
is recorded — an "audit-write-only" posture you would tighten into an explicit
allow-list once you know the workload's real syscall footprint.

**The trap:** a pod that references a Localhost profile the node doesn't have
fails to start with `CreateContainerError`. That is why the file is placed on the
node *before* the workload is deployed. (The gym recreates the cluster on every
`start`, wiping the node filesystem — so setup re-places the file each time.)

## Why Unconfined is dangerous

`Unconfined` installs no seccomp filter, so the container can invoke **any**
syscall the kernel offers. That hands an attacker who lands in the container the
full syscall surface — namespace creation (`unshare`), `keyctl`, `ptrace`,
obscure/legacy calls — the exact primitives used to escalate and break out.
Under `RuntimeDefault`, `unshare --map-root-user --user true` returns
`Operation not permitted`; under `Unconfined` it succeeds. That one-line
difference is seccomp doing its job.

## Verify by hand

```bash
# structural
kubectl -n public     get deploy web-frontend -o jsonpath='{.spec.template.spec.securityContext.seccompProfile.type}'; echo
kubectl -n monitoring get deploy log-writer   -o jsonpath='{.spec.template.spec.securityContext.seccompProfile}{"\n"}'

# both still running
kubectl -n public get deploy web-frontend; kubectl -n monitoring get deploy log-writer

# the dangerous syscall is now blocked in web-frontend (RuntimeDefault):
kubectl -n public exec deploy/web-frontend -- unshare --map-root-user --user true
#   -> unshare: Operation not permitted   (non-zero) — seccomp working
```
