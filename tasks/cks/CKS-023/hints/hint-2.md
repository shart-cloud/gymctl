# CKS-023 Check Criteria

- Audit policy file exists with correct order
- None for healthz/readyz
- Metadata for secrets, RequestResponse for pods/exec, Request for changes
- Catch-all Metadata
- API server has audit flags and is healthy
- Audit log contains events at correct levels
