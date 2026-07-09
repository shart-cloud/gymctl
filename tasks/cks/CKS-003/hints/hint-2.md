# CKS-003 Check Criteria

- dashboard-tls secret exists with cert/key
- Ingress tls block references dashboard-tls for dashboard.lab.local
- No cluster-admin binding remains for dashboard SA
- dashboard-viewer ClusterRole has get/list/watch on required resources
- ClusterRoleBinding binds dashboard-viewer to dashboard SA
- Dashboard is running and accessible
