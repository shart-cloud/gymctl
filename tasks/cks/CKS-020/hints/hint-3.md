# CKS-020 — Hint 3 (cost 50)

Near-complete manifest. The `caBundle` below is the fixed lab CA (base64) — the
full, ready-to-apply file is in `solution/image-policy-webhook.yaml`.

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: image-policy-webhook
webhooks:
  - name: image-policy.policy-system.svc
    admissionReviewVersions: ["v1"]
    sideEffects: None
    failurePolicy: Fail
    rules:
      - apiGroups: [""]
        apiVersions: ["v1"]
        operations: ["CREATE", "UPDATE"]
        resources: ["pods"]
        scope: "Namespaced"
    clientConfig:
      service:
        namespace: policy-system
        name: image-policy
        path: /validate
        port: 8443
      caBundle: <base64 of the lab CA — see solution/image-policy-webhook.yaml>
    namespaceSelector:
      matchExpressions:
        - key: kubernetes.io/metadata.name
          operator: NotIn
          values: ["kube-system", "policy-system"]
```

Get the exact caBundle string from `solution/image-policy-webhook.yaml`, or
derive it yourself — it is the base64 DER/PEM of the CA that signed the Secret
`policy-system/image-policy-tls`. You can sanity-check the pairing:

```bash
# CA that must go in caBundle vs the serving cert it has to validate:
kubectl -n policy-system get secret image-policy-tls -o jsonpath='{.data.tls\.crt}' \
  | base64 -d | openssl x509 -noout -issuer -subject
```

Apply, then verify with `--dry-run=server` creates (see hint 2). Full manifest
in `solution/`.
