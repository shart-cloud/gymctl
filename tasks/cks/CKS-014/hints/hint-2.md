# CKS-014 Check Criteria

- No backend ConfigMap contains passwords
- db-credentials Secret exists with correct data
- api-server uses secretKeyRef
- db-backup mounts Secret and has no password args
- API server references EncryptionConfiguration
- aescbc key is correct
- API server healthy and etcd data is encrypted
