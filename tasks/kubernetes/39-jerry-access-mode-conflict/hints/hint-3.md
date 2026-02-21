Delete the existing resources and recreate with the correct access mode:

```bash
kubectl delete pod jerry-rwx-app --ignore-not-found
kubectl delete pvc jerry-rwx-claim --ignore-not-found

kubectl apply -f - <<'EOF'
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: jerry-rwx-claim
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: local-rwo
EOF

kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: jerry-rwx-app
spec:
  containers:
  - name: app
    image: busybox:1.36
    command: ["sh", "-c", "echo running && sleep 3600"]
    volumeMounts:
    - name: data
      mountPath: /data
  volumes:
  - name: data
    persistentVolumeClaim:
      claimName: jerry-rwx-claim
EOF
```

Verify:

```bash
kubectl get pvc jerry-rwx-claim
kubectl get pod jerry-rwx-app
```
