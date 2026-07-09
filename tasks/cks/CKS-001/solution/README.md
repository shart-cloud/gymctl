# CKS-001 — Solution

Apply all three policies:

```bash
kubectl apply -f solution/default-deny.yaml
kubectl apply -f solution/allow-payment-api.yaml
kubectl apply -f solution/allow-payment-db.yaml
```

## Why each piece is needed

| Policy | Selects | Purpose |
| --- | --- | --- |
| `default-deny` | all pods (`podSelector: {}`) | Baseline: drop all ingress + egress in `payments`. |
| `allow-payment-api` | `app=payment-api` | Ingress from `web/app=frontend:8080`; egress to `payment-db:5432` and to kube-dns on 53. |
| `allow-payment-db` | `app=payment-db` | Ingress from `app=payment-api:5432` — the other half of the API→DB path. |

## The two traps

1. **DNS.** A default-deny *egress* baseline silently breaks name resolution.
   Without the UDP/TCP 53 egress rule to `kube-system/k8s-app=kube-dns`,
   `payment-api` can't resolve `payment-db` and the DB egress rule never fires.
2. **Both ends enforce.** NetworkPolicy is applied at the egress side of the
   source *and* the ingress side of the destination. Allowing `payment-api` to
   egress to the DB is not enough — `payment-db` also needs an ingress allow, or
   the packet is dropped on arrival. That's what `allow-payment-db` fixes.

## Verify by hand

```bash
# allowed
kubectl -n web      exec deploy/frontend    -- /agnhost connect payment-api.payments.svc.cluster.local:8080 --timeout=6s
kubectl -n payments exec deploy/payment-api -- /agnhost connect payment-db.payments.svc.cluster.local:5432  --timeout=6s
# denied (these should time out / fail)
kubectl -n payments exec deploy/payment-api -- /agnhost connect 1.1.1.1:443 --timeout=5s
kubectl -n web      exec deploy/prober      -- /agnhost connect payment-api.payments.svc.cluster.local:8080 --timeout=5s
```
