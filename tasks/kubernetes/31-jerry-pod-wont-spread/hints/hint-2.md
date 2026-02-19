# Hint 2: Scheduling Constraints

You need to add scheduling constraints to spread pods across nodes. Two approaches:

**podAntiAffinity**: Prevents pods with the same label from being on the same node
**topologySpreadConstraints**: Distributes pods evenly across a topology domain

Use `kubectl edit deployment jerry-ha-app` to add constraints to the pod spec.

The topology key `kubernetes.io/hostname` represents individual nodes.