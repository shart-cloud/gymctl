# Hint 2: Node Label Mismatch

Jerry's deployment requires nodes with `hardware-type=gpu-enabled` but no nodes have this label.

Check current node labels:
- `kubectl get nodes --show-labels`

You can either:
1. Add the required label to a node: `kubectl label nodes <node-name> hardware-type=gpu-enabled`
2. Update the deployment's affinity to match an existing label

The required affinity is in `.spec.template.spec.affinity.nodeAffinity`