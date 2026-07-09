# CKS-001 — Hint 2 (cost 25)

You need three policies in `payments`:

1. `default-deny` — `podSelector: {}`, `policyTypes: [Ingress, Egress]`, no rules.
2. `allow-payment-api` (selects `app=payment-api`):
   - Ingress from `web` namespace + `app=frontend`, port 8080.
   - Egress to `app=payment-db`, port 5432.
   - Egress to kube-dns on UDP **and** TCP 53.
3. `allow-payment-db` (selects `app=payment-db`):
   - Ingress from `app=payment-api`, port 5432.

Two things that trip people up:

- **DNS first.** If the API can't reach CoreDNS, it can't resolve
  `payment-db` and nothing else works. The DNS peer is in `kube-system` with
  label `k8s-app=kube-dns`.
- **Both ends.** Allowing the API to *egress* to the DB isn't enough — the DB
  also needs an *ingress* rule accepting the API. That's why `allow-payment-db`
  exists.

Select the `web` namespace with the automatic label
`kubernetes.io/metadata.name: web`.
