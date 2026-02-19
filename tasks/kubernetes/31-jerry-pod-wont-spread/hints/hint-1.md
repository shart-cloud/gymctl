# Hint 1: Pod Distribution Problem

Jerry's pods are all clustering on the same node, which defeats high availability.

Check the current pod placement:
- `kubectl get pods -l app=jerry-ha-app -o wide`
- Look at the NODE column to see if they're all on one node

The deployment needs constraints to spread pods across different nodes for resilience.