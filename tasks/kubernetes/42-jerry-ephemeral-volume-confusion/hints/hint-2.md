Replace the `emptyDir` with a `PersistentVolumeClaim`.

First create the PVC:

```bash
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: jerry-upload-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: fast-local
EOF
```

Then delete the existing pod and recreate it with the PVC instead of the emptyDir.
The `volumes` section needs to change from:

```yaml
volumes:
- name: upload-storage
  emptyDir: {}
```

to:

```yaml
volumes:
- name: upload-storage
  persistentVolumeClaim:
    claimName: jerry-upload-pvc
```
