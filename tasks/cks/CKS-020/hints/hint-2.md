# CKS-020 Check Criteria

- ValidatingWebhookConfiguration exists
- Webhook targets CREATE/UPDATE pods
- CA bundle is set
- namespaceSelector excludes kube-system and policy-system
- failurePolicy is Fail
- Docker Hub image rejected
- Trusted explicit tag succeeds
- latest tag rejected
