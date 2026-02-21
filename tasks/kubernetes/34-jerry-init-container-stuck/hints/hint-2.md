# Hint 2: Compare the ConfigMap Key Against the Env Reference

The init container reads `DB_MIGRATION_SCRIPT` from a ConfigMap key reference.
Check what key the Deployment references versus what keys actually exist in the
ConfigMap:

```bash
kubectl get configmap jerry-migration-config -o yaml
kubectl get deployment jerry-init-app \
  -o jsonpath='{.spec.template.spec.initContainers[0].env}'
```

Do the `key` values match exactly?
