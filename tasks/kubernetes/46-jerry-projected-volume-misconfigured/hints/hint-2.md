Projected volume sources use different field names than `envFrom` or `env.valueFrom`.

| Wrong (envFrom style) | Correct (projected source style) |
|---|---|
| `configMapRef:` | `configMap:` |
| `secretRef:` | `secret:` |

The `*Ref` variants are for pulling values into environment variables, not file mounts.

For projected volumes, the correct structure is:

```yaml
projected:
  sources:
  - configMap:
      name: my-configmap
      items:
      - key: my-key
        path: filename-in-mount
  - secret:
      name: my-secret
      items:
      - key: my-key
        path: filename-in-mount
```
