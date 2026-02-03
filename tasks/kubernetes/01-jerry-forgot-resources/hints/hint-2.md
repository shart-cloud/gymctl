## Hint 2: Requests vs Limits

Kubernetes uses two types of resource specifications:

**Requests**: The guaranteed amount of resources for the container.
**Limits**: The maximum amount the container can use.

Pods without requests or limits are "BestEffort" and are first to be evicted.
