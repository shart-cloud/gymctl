# CKS-006 Check Criteria

- default SA automountServiceAccountToken=false
- web-app and worker disable token automount
- api-sa exists with minimal Role/RoleBinding
- api-server uses api-sa
- Token file absent in web-app/worker and present in api-server
