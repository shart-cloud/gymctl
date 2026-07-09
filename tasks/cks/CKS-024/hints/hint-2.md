# CKS-024 Check Criteria

- All three deployments have readOnlyRootFilesystem=true
- Each has emptyDir volumes for write paths
- allowPrivilegeEscalation=false
- All pods Running
- Writes to /etc or /usr fail and designated dirs succeed
