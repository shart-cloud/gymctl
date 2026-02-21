The local PV has nodeAffinity requiring a specific worker node. That node is cordoned —
marked `SchedulingDisabled`. The pod's implicit requirement (via the PVC → PV → nodeAffinity
chain) forces it to schedule on that exact node, which won't accept new pods.

Two ways to fix this:

**Option A (simpler):** Uncordon the target node so the pod can schedule there.

```bash
TARGET=$(kubectl get nodes -l jerry.gym/role=cordoned-target -o jsonpath='{.items[0].metadata.name}')
kubectl uncordon "$TARGET"
```

**Option B (migration approach):** Update the PV's nodeAffinity to point to the other
available worker. This requires:
1. Creating the storage directory on the new node
2. Patching the PV nodeAffinity
3. Restarting the pod

Note: Option B does not move your data — for a real migration you'd need to copy data
between nodes before switching the affinity.
