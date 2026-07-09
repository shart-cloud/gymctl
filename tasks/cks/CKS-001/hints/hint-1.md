# CKS-001 — Hint 1 (free)

Start by seeing what's there and what talks to what:

```bash
kubectl -n payments get pods,svc,networkpolicy
kubectl -n web get pods --show-labels
```

There are no NetworkPolicies yet, so everything is allowed. Two ideas drive the
whole exercise:

- A **default-deny** policy is just an empty `podSelector: {}` plus
  `policyTypes: [Ingress, Egress]` and **no** rules.
- Once you deny by default, you must *explicitly* re-allow every path you need —
  including **DNS**.

Docs: <https://kubernetes.io/docs/concepts/services-networking/network-policies/>
