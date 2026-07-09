# CKS-012 Check Criteria

- Pod securityContext uses UID/GID 1000 and runAsNonRoot
- Container readOnlyRootFilesystem=true
- Capabilities drop ALL and add NET_BIND_SERVICE
- allowPrivilegeEscalation=false
- emptyDir mounted at /tmp
- Pod Running; /etc write fails and /tmp write succeeds
