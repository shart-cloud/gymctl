# CKS-001 — Hint 3 (cost 50)

Default deny:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: { name: default-deny, namespace: payments }
spec:
  podSelector: {}
  policyTypes: [Ingress, Egress]
```

API policy (ingress from frontend + egress to DB and DNS):

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: { name: allow-payment-api, namespace: payments }
spec:
  podSelector: { matchLabels: { app: payment-api } }
  policyTypes: [Ingress, Egress]
  ingress:
    - from:
        - namespaceSelector: { matchLabels: { kubernetes.io/metadata.name: web } }
          podSelector: { matchLabels: { app: frontend } }
      ports: [ { protocol: TCP, port: 8080 } ]
  egress:
    - to: [ { podSelector: { matchLabels: { app: payment-db } } } ]
      ports: [ { protocol: TCP, port: 5432 } ]
    - to:
        - namespaceSelector: { matchLabels: { kubernetes.io/metadata.name: kube-system } }
          podSelector: { matchLabels: { k8s-app: kube-dns } }
      ports: [ { protocol: UDP, port: 53 }, { protocol: TCP, port: 53 } ]
```

DB ingress:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: { name: allow-payment-db, namespace: payments }
spec:
  podSelector: { matchLabels: { app: payment-db } }
  policyTypes: [Ingress]
  ingress:
    - from: [ { podSelector: { matchLabels: { app: payment-api } } } ]
      ports: [ { protocol: TCP, port: 5432 } ]
```

The full manifests are in `solution/`.
