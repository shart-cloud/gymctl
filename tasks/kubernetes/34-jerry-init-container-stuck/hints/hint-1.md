# Hint 1: Read the Init Container's Output

The pod is in `Init:CrashLoopBackOff` — start by reading what the init container
actually printed before it exited.

```bash
kubectl get pods -l app=jerry-init-app
kubectl logs -l app=jerry-init-app -c db-migration --previous
```

What does the error message say? That tells you exactly what the init container
couldn't find.
