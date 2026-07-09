# CKS-005 — Hint 2 (cost 25)

Four objects, plus one deletion:

1. `kubectl delete clusterrolebinding ci-bot-cluster-admin`
2. `Role/ci-bot-dev` in `dev-team`: verbs
   `get,list,watch,create,update,patch,delete` on `deployments` (apps group),
   `services`, `configmaps` (core group).
3. `RoleBinding/ci-bot-dev` → subject SA `dev-team/ci-bot`.
4. `ClusterRole/ci-bot-readonly`: `get,list` on `namespaces` and `nodes` only.
5. `ClusterRoleBinding/ci-bot-readonly` → same subject.

Gotchas:
- `deployments` are in `apiGroups: ["apps"]`; `services`/`configmaps` are in the
  core group `apiGroups: [""]`.
- Do **not** add `secrets` anywhere, and never use `verbs: ["*"]`.
