# CKS-013 — Hint 3 (cost 50)

Deny-privileged template + constraint:

```yaml
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata: { name: k8sdenyprivileged }
spec:
  crd:
    spec:
      names: { kind: K8sDenyPrivileged }
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package k8sdenyprivileged
        violation[{"msg": msg}] {
          c := input.review.object.spec.containers[_]
          c.securityContext.privileged == true
          msg := sprintf("privileged container is not allowed: %v", [c.name])
        }
---
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: K8sDenyPrivileged
metadata: { name: deny-privileged }
spec:
  match:
    kinds: [ { apiGroups: [""], kinds: ["Pod"] } ]
    excludedNamespaces: ["kube-system"]
```

Required-labels template + constraint (parameterized):

```yaml
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata: { name: k8srequiredlabels }
spec:
  crd:
    spec:
      names: { kind: K8sRequiredLabels }
      validation:
        openAPIV3Schema:
          type: object
          properties:
            labels: { type: array, items: { type: string } }
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package k8srequiredlabels
        violation[{"msg": msg}] {
          required := input.parameters.labels[_]
          not input.review.object.metadata.labels[required]
          msg := sprintf("missing required label: %v", [required])
        }
---
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: K8sRequiredLabels
metadata: { name: require-owner-label }
spec:
  match:
    kinds: [ { apiGroups: [""], kinds: ["Pod"] } ]
    namespaces: ["app", "staging"]
  parameters:
    labels: ["owner"]
```

Apply templates, `kubectl wait --for condition=established crd/...`, then the
constraints (or `kubectl apply -f solution/` twice). The full manifests are in
`solution/`.
