# Hint 2: Cross-namespace service discovery

Services in different namespaces need fully qualified domain names (FQDN):

Format: `<service>.<namespace>.svc.cluster.local`

For the backend service:
- Service name: backend-api
- Namespace: backend
- Full name: backend-api.backend.svc.cluster.local

Update the BACKEND_URL environment variable in the frontend deployment.