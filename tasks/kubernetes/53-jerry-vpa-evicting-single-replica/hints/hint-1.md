# Hint 1: Check the VPA Update Mode

VPA evicts running pods only when its updateMode allows it.

Check what mode is currently set:

```bash
kubectl describe vpa jerry-singleton-vpa
```

Look for `Update Policy` → `Update Mode`. What does it say?

Then look at the Kubernetes docs:
```bash
kubectl explain vpa.spec.updatePolicy.updateMode
```

Which mode would apply recommendations only to new pods — never to running ones?
