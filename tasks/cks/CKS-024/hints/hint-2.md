# CKS-024 — Hint 2 (cost 25)

Apply this block to `web`, `api`, **and** `cache`:

```yaml
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
          volumeMounts:
            - { name: tmp, mountPath: /tmp }
      volumes:
        - { name: tmp, emptyDir: {} }
```

`securityContext` and `volumeMounts` go on the container; `volumes` goes on the
pod spec (sibling of `containers`).

If a deployment won't become Available after the change, it's writing somewhere
other than `/tmp` — check its logs and mount an emptyDir there too.
