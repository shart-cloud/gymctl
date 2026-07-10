# Hint 3 — Near-complete manifests (cost 50)

Patch each Deployment's pod template. Only the `seccompProfile` changes; keep
everything else (image, command, name, namespace) the same.

`web-frontend` (namespace `public`):

```yaml
spec:
  template:
    spec:
      securityContext:
        seccompProfile:
          type: RuntimeDefault
```

`log-writer` (namespace `monitoring`):

```yaml
spec:
  template:
    spec:
      securityContext:
        seccompProfile:
          type: Localhost
          localhostProfile: profiles/audit-write-only.json
```

Apply and confirm:

```bash
kubectl apply -f <your-edited-manifests>
kubectl -n public     rollout status deploy/web-frontend
kubectl -n monitoring rollout status deploy/log-writer

# the negative proof — this must FAIL under RuntimeDefault:
kubectl -n public exec deploy/web-frontend -- unshare --map-root-user --user true
#   -> unshare: Operation not permitted
```

The complete manifests are in `solution/` (`01-web-frontend-runtimedefault.yaml`,
`02-log-writer-localhost.yaml`), and `solution/README.md` explains RuntimeDefault
vs Localhost, where the profile lives, and why Unconfined is dangerous.
