# CKS-006 — Hint 2 (cost 25)

Do all four:

1. Patch the default SA:
   `kubectl -n production patch sa default -p '{"automountServiceAccountToken":false}'`
2. Add `automountServiceAccountToken: false` to the **pod template spec** of
   `web-app` and `worker` (under `spec.template.spec`, a sibling of
   `containers`). Editing the deployment rolls new pods with no token file.
3. Create `api-sa` + a `Role` (`get`/`list` on `secrets`,`configmaps`) +
   `RoleBinding`.
4. Set `spec.template.spec.serviceAccountName: api-sa` on the `api-server`
   deployment; leave its automount on (default).

Remember: the token file lives at
`/var/run/secrets/kubernetes.io/serviceaccount/token`.
