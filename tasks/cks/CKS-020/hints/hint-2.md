# CKS-020 — Hint 2 (cost 25)

One object: `ValidatingWebhookConfiguration` `image-policy-webhook`, with a
single webhook entry. The pieces:

## clientConfig — reach the backend
- `service:` `namespace: policy-system`, `name: image-policy`, `path: /validate`,
  `port: 8443`.
- `caBundle:` the **base64 CA** that signed the backend's serving cert. It is not
  the same as the cert in the Secret — it is the CA. Grab it from the exercise's
  solution/hint-3, or in a real setup from wherever you issued the certs. A
  wrong or empty caBundle fails the TLS handshake, and with `failurePolicy: Fail`
  that blocks *every* pod create.

## rules — which requests to intercept
```yaml
rules:
  - apiGroups: [""]
    apiVersions: ["v1"]
    operations: ["CREATE", "UPDATE"]
    resources: ["pods"]
```

## The three "don't brick the cluster" details
1. `failurePolicy: Fail` — required, and the reason the next two matter.
2. `namespaceSelector` must exclude **both** `kube-system` (control plane) and
   `policy-system` (the backend's own namespace — or a backend restart deadlocks
   the webhook against itself). Use a `matchExpressions` `NotIn` on the built-in
   `kubernetes.io/metadata.name` label:
   ```yaml
   namespaceSelector:
     matchExpressions:
       - key: kubernetes.io/metadata.name
         operator: NotIn
         values: ["kube-system", "policy-system"]
   ```
3. `sideEffects: None` and `admissionReviewVersions: ["v1"]` are required fields
   for a v1 webhook.

## Test without creating anything
`kubectl apply --dry-run=server -f -` (or `kubectl run --dry-run=server`) invokes
the webhook but persists nothing:
- `nginx:latest` and `nginx:1.27` in `default` → **rejected**.
- `registry.k8s.io/pause:3.9` in `default` → **admitted**.
- `nginx:latest` in `policy-system` → **admitted** (proves the exclusion).

There is a few-second delay between applying the config and the webhook actually
enforcing — retry if your first test slips through.
