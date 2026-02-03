## Hint 3: Add Resource Limits

Add a `resources` block to the container spec:

```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "256Mi"
    cpu: "200m"
```
