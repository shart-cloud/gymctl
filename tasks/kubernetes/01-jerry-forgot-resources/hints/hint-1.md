## Hint 1: Investigate the Node

Start by examining the node's resource allocation:

```bash
kubectl describe node | grep -A 20 "Allocated resources"
```

What do you notice about how resources are distributed?
