# Hint 3: Patch It Directly

```bash
kubectl patch vpa jerry-singleton-vpa --type=merge -p='{
  "spec": {
    "updatePolicy": {
      "updateMode": "Initial"
    }
  }
}'
```

Verify:

```bash
kubectl get vpa jerry-singleton-vpa \
  -o jsonpath='{.spec.updatePolicy.updateMode}'
```

Should output `Initial`. VPA will no longer evict the running pod — it will apply recommendations the next time the pod is naturally replaced (deployment rollout, node drain, etc.).
