# CKS-024 — Solution

```bash
kubectl apply -f solution/workloads.yaml
```

Same pattern applied to all three deployments (`web`, `api`, `cache`):

```yaml
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
          volumeMounts:
            - { name: tmp, mountPath: /tmp }
      volumes:
        - { name: tmp, emptyDir: {} }
```

## Why

- **`readOnlyRootFilesystem: true`** makes the container image immutable at
  runtime — an attacker can't drop a binary or patch a config in the image
  layers.
- **`emptyDir` at `/tmp`** gives the process the writable scratch space it still
  needs. Without it, anything that writes to disk crash-loops and the
  "Available" check fails. In the real world you'd mount one emptyDir per write
  path the app actually uses.
- **`allowPrivilegeEscalation: false`** stops a process from gaining more
  privileges than its parent (setuid binaries, etc.).

## Verify

```bash
for d in web api cache; do
  kubectl -n production exec deploy/$d -- sh -c 'echo ok > /tmp/x'   # succeeds
  kubectl -n production exec deploy/$d -- sh -c 'echo x > /etc/x'    # fails: read-only
done
```
