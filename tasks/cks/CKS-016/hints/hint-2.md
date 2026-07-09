# CKS-016 — Hint 2 (cost 25)

You need three labels total:

- `restricted-apps`: `pod-security.kubernetes.io/enforce=restricted`
- `baseline-apps`: `pod-security.kubernetes.io/enforce=baseline`
- `baseline-apps`: `pod-security.kubernetes.io/warn=restricted`

```bash
kubectl label ns restricted-apps pod-security.kubernetes.io/enforce=restricted --overwrite
kubectl label ns baseline-apps  pod-security.kubernetes.io/enforce=baseline    --overwrite
kubectl label ns baseline-apps  pod-security.kubernetes.io/warn=restricted     --overwrite
```

A plain busybox pod with no `securityContext` violates `restricted` (not
runAsNonRoot, caps not dropped, no seccomp) but satisfies `baseline` — that's
why it's rejected in one namespace and admitted in the other.
