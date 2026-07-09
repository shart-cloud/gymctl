# CKS-012 — Hint 2 (cost 25)

Pod-level `securityContext` (sibling of `containers`):

```yaml
      securityContext:
        runAsUser: 1000
        runAsGroup: 1000
        runAsNonRoot: true
```

Container-level `securityContext`:

```yaml
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            capabilities:
              drop: ["ALL"]
              add: ["NET_BIND_SERVICE"]
```

And because the root fs is now read-only, add scratch space:

```yaml
          volumeMounts:
            - { name: tmp, mountPath: /tmp }
      volumes:
        - { name: tmp, emptyDir: {} }
```

If the pod won't become Available, it's almost always the read-only root with no
writable mount for where it writes.
