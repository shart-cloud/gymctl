# Hint 3: Nested Image Paths and Sort Order

For QUERY3, container images live inside `.spec.containers[*]` inside each pod:

```bash
kubectl get pods -l app=jerry-api -o jsonpath='{.items[*].spec.containers[*].image}'
```

For QUERY4, add `--sort-by` to order pods by creation timestamp:

```bash
kubectl get pods -l app=jerry-api \
  --sort-by=.metadata.creationTimestamp \
  -o jsonpath='{.items[*].metadata.name}'
```

Edit `/tmp/jerry-jsonpath-broken.sh` with these corrected expressions. The
variable names (`QUERY1`–`QUERY4`) must stay the same.
