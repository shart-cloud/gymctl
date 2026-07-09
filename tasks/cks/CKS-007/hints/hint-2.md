# CKS-007 Check Criteria

- anonymous-auth=false
- enable-admission-plugins includes NodeRestriction and PodSecurity
- kubelet-certificate-authority points to CA
- tls-min-version=VersionTLS12
- insecure-port absent or 0
- API server is running and kubectl works
