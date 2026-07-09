# CKS-020 — Hint 1 (free)

Get the lay of the land:

```bash
kubectl get validatingwebhookconfigurations           # nothing named image-policy-webhook
kubectl -n policy-system get deploy,svc,secret,cm      # the backend is already here and Available
kubectl -n default get deploy legacy-nginx -o jsonpath='{..image}'; echo   # nginx:latest, admitted
```

The admission **backend** is already built and running: a Service
`image-policy` in `policy-system` on port `8443` (path `/validate`) that speaks
AdmissionReview v1 and enforces the trusted-image policy. It enforces nothing
yet because nothing tells the API server to call it.

You author exactly one object: a **ValidatingWebhookConfiguration** named
`image-policy-webhook` whose `clientConfig` points at that Service, with the
right caBundle, rules, exclusions, and failure policy.

The trap that makes this hard: `failurePolicy: Fail` means a *wrong* config
(bad caBundle, or forgetting to exclude the backend's own namespace) doesn't
just fail this exercise — it blocks pod creation cluster-wide. Get the three
"don't brick it" details right (see hint 2).

Docs: <https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/>
