# Hint 2: List Traversal and Field Name Case

Two common jsonpath mistakes are present in this script:

1. **List traversal**: `{.items.metadata.name}` does not iterate — you need
   `{.items[*].metadata.name}` or a `range` expression.

2. **Case sensitivity**: jsonpath field names are case-sensitive. Check whether
   `.spec.nodename` should be `.spec.nodeName`.

Test expressions directly to verify:

```bash
kubectl get pods -l app=jerry-api -o jsonpath='{.items[*].metadata.name}'
kubectl get pod jerry-api-0 -o jsonpath='{.spec.nodeName}'
```
