# Hint 3: The Fix

**Option A - Label a node:**
```bash
kubectl get nodes
kubectl label nodes <worker-node-name> hardware-type=gpu-enabled
```

**Option B - Fix the affinity rule:**
Edit the deployment to use an existing label like `kubernetes.io/arch=amd64`:
```bash
kubectl edit deployment jerry-picky-app
```

Change the affinity to:
```yaml
nodeAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
    nodeSelectorTerms:
    - matchExpressions:
      - key: kubernetes.io/arch
        operator: In
        values:
        - amd64
```

Either approach satisfies the requirement to keep an affinity constraint while making the pod schedulable.