# Hint 1: Scheduling Failure

Jerry's pod is stuck in Pending state because of an unmatched scheduling constraint.

Check:
- `kubectl get pods -l app=jerry-picky-app`
- `kubectl describe pod <pod-name>` - look at the Events section

The scheduler can't find nodes that satisfy the nodeAffinity requirement.