# Hint 1: Understand Why They Conflict

HPA calculates utilization like this:

```
utilization % = (actual CPU usage) / (requests.cpu) * 100
```

When VPA changes `requests.cpu`, the denominator shifts. Lower requests → higher apparent utilization → HPA scales out. VPA then raises requests → lower apparent utilization → HPA scales in. Loop forever.

Inspect both objects:

```bash
kubectl describe hpa jerry-app-hpa
kubectl describe vpa jerry-app-vpa
```

What resource is HPA targeting? What resources is VPA currently controlling?
