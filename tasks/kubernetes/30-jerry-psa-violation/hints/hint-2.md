# Hint 2: Restricted Security Standards

The `restricted` Pod Security Standard requires:

- `runAsNonRoot: true` and `runAsUser: <non-zero>`
- `allowPrivilegeEscalation: false`
- `capabilities.drop: ["ALL"]`
- `seccompProfile.type: RuntimeDefault`
- `readOnlyRootFilesystem: true` (recommended)

Jerry's deployment violates multiple requirements. Check the deployment YAML and update the `securityContext` at both pod and container level.

Use `kubectl describe deployment jerry-app -n production` to see the admission errors.