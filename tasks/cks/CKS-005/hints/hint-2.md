# CKS-005 Check Criteria

- No cluster-admin binding remains for ci-bot
- Role ci-bot-dev has correct namespace rules
- RoleBinding binds ci-bot-dev to ci-bot
- ClusterRole ci-bot-readonly has get/list on namespaces and nodes only
- ClusterRoleBinding binds ci-bot-readonly to ci-bot
- can-i create deployments yes; delete pods and get secrets no
