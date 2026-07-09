# CKS-016 — Solution

Label-only. Either apply the manifest or use `kubectl label`:

```bash
kubectl apply -f solution/namespaces-labeled.yaml
```

or

```bash
kubectl label ns restricted-apps pod-security.kubernetes.io/enforce=restricted --overwrite
kubectl label ns baseline-apps  pod-security.kubernetes.io/enforce=baseline    --overwrite
kubectl label ns baseline-apps  pod-security.kubernetes.io/warn=restricted     --overwrite
```

## How PSA labels work

`pod-security.kubernetes.io/<MODE>: <LEVEL>` where:
- **MODE** = `enforce` (reject), `warn` (client warning), `audit` (audit log).
- **LEVEL** = `privileged` (no restrictions), `baseline` (block known escalations),
  `restricted` (hardened: non-root, drop caps, seccomp, no privilege escalation).

`baseline-apps` uses two modes at once: `enforce: baseline` blocks the dangerous
stuff, and `warn: restricted` nudges owners toward the stricter bar without
breaking them yet — the standard rollout pattern.

## Verify

```bash
# rejected under restricted, admitted under baseline:
kubectl -n restricted-apps run t --image=busybox:1.36 --dry-run=server -- sleep 1   # error
kubectl -n baseline-apps  run t --image=busybox:1.36 --dry-run=server -- sleep 1    # ok (with warning)
```
