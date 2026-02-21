# Hint 3: Edit the Deployment to Fix the Key Name

The `configMapKeyRef.key` in the Deployment must exactly match a key that exists
in the ConfigMap. Edit the Deployment to correct it:

```bash
kubectl edit deployment jerry-init-app
```

Find `configMapKeyRef` inside the `initContainers` section and change the `key`
field to match the key that actually exists in `jerry-migration-config`.

After saving, watch the pod restart: `kubectl get pods -l app=jerry-init-app -w`
