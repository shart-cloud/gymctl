# Hint 2 — The objects and the traps (cost 25)

Set the **pod-level** `securityContext.seccompProfile` on each Deployment
(`.spec.template.spec.securityContext.seccompProfile`):

- **`web-frontend` → RuntimeDefault**

  ```yaml
  securityContext:
    seccompProfile:
      type: RuntimeDefault
  ```

  RuntimeDefault is the runtime's built-in profile — nothing to place on the
  node. It blocks dangerous syscalls (e.g. `unshare`) while allowing normal app
  syscalls.

- **`log-writer` → Localhost**, referencing `profiles/audit-write-only.json`

  ```yaml
  securityContext:
    seccompProfile:
      type: Localhost
      localhostProfile: profiles/audit-write-only.json
  ```

  `localhostProfile` is resolved relative to the kubelet seccomp root
  `/var/lib/kubelet/seccomp/`, so this points at
  `/var/lib/kubelet/seccomp/profiles/audit-write-only.json`.

Traps:

1. **The profile file already exists on the node** — the exercise placed
   `/var/lib/kubelet/seccomp/profiles/audit-write-only.json` for you. You do
   **not** need to create it. (A pod that references a *missing* Localhost
   profile fails to start with `CreateContainerError`.)
2. **Put it at pod level**, not on the container, and don't add extra fields —
   the checks read `.spec.template.spec.securityContext.seccompProfile`.
3. **Both Deployments must stay Available.** Changing the profile rolls the pods;
   confirm they come back Running.

Apply your edited Deployments with `kubectl apply -f`.
