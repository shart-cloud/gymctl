# CKS-002 Check Criteria

- API server manifest has anonymous-auth=false, profiling=false, audit-log-path configured
- Kubelet config disables anonymous auth, uses Webhook authorization, and protectKernelDefaults=true
- API server and kubelet restart healthy
- kube-bench targeted checks pass
