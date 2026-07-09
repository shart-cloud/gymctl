# CKS-013 — Hint 2 (cost 25)

Four objects, two templates and two constraints.

## Rule 1 — deny privileged

- **ConstraintTemplate** `k8sdenyprivileged` defining kind `K8sDenyPrivileged`.
  The Rego iterates `input.review.object.spec.containers[_]` (and
  `initContainers[_]`) and emits a `violation` when
  `c.securityContext.privileged == true`.
- **Constraint** `deny-privileged` of that kind, matching `Pod`, with
  `match.excludedNamespaces: [kube-system]`.

## Rule 2 — require the owner label

- **ConstraintTemplate** `k8srequiredlabels` defining kind `K8sRequiredLabels`.
  Make the label key(s) a **parameter** — the Rego reads
  `input.parameters.labels[_]` and violates when
  `not input.review.object.metadata.labels[required]`. Declare the parameter in
  the template's `validation.openAPIV3Schema`.
- **Constraint** `require-owner-label` of that kind, matching `Pod`, with
  `match.namespaces: [app, staging]` and `parameters.labels: [owner]`.

## Traps

- **Order.** A Constraint's CRD is created by its template. Apply templates
  first, wait for the CRDs, then the constraints — or you'll see
  `no matches for kind "K8sDenyPrivileged"`.
- **Scope on the Constraint, not the Rego.** `match.namespaces` opts specific
  namespaces in; `match.excludedNamespaces` carves them out. Keep the Rego
  namespace-agnostic.
- **Propagation delay.** Give the webhook a few seconds after creating a
  Constraint before you trust a test pod's result.

Test with `kubectl apply --dry-run=server -f -` (it invokes the webhook without
creating anything).
