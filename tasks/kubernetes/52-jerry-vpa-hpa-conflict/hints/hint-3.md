# Hint 3: Patch It Directly

If you prefer a non-interactive fix:

```bash
kubectl patch vpa jerry-app-vpa --type=merge -p='{
  "spec": {
    "resourcePolicy": {
      "containerPolicies": [{
        "containerName": "app",
        "controlledResources": ["memory"]
      }]
    }
  }
}'
```

Verify the change:

```bash
kubectl get vpa jerry-app-vpa \
  -o jsonpath='{.spec.resourcePolicy.containerPolicies[0].controlledResources}'
```

Should output `["memory"]`. CPU is now out of VPA's scope — HPA can manage it without interference.
