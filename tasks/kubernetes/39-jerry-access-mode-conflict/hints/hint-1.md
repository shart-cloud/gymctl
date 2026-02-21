Start with the PVC status and events:

```bash
kubectl get pvc jerry-rwx-claim
kubectl describe pvc jerry-rwx-claim
kubectl get events --sort-by=.metadata.creationTimestamp | tail -20
```

Look for a provisioning or access mode error in the events.

Then check what the StorageClass supports:

```bash
kubectl get storageclass local-rwo -o yaml
```

Local-path provisioner only supports ReadWriteOnce — it's node-local storage.
ReadWriteMany requires a network-backed shared filesystem (NFS, CephFS, etc.).
