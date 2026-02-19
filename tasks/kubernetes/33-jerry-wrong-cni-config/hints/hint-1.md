# Hint 1: CNI Networking Issue

Jerry has broken the CNI (Container Network Interface) configuration, causing networking problems.

Look for:
- Pods stuck in ContainerCreating state
- Network-related errors in pod events
- CNI system pods that might be missing or failing

Check the kube-system namespace for CNI-related pods and their status.