# CKS-013 Check Criteria

- Deny privileged ConstraintTemplate exists with valid Rego
- Constraint excludes kube-system
- Required labels template supports parameterized key
- Constraint scopes to app and staging
- Privileged pod and unlabeled app pod fail
- Compliant pod succeeds
