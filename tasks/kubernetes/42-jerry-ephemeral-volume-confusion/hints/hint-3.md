Full replacement:

```bash
kubectl delete pod jerry-uploader --ignore-not-found

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

kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: jerry-uploader
  labels:
    app: jerry-uploader
spec:
  containers:
  - name: uploader
    image: busybox:1.36
    command: ["sh", "-c", "echo started > /uploads/init.log && sleep 3600"]
    volumeMounts:
    - name: upload-storage
      mountPath: /uploads
  volumes:
  - name: upload-storage
    persistentVolumeClaim:
      claimName: jerry-upload-pvc
EOF

kubectl wait --for=condition=Ready pod/jerry-uploader --timeout=60s
```

Test persistence:

```bash
kubectl exec jerry-uploader -- sh -c 'echo proof > /uploads/sentinel'
kubectl delete pod jerry-uploader
kubectl apply -f <your-pod-manifest>
kubectl wait --for=condition=Ready pod/jerry-uploader --timeout=60s
kubectl exec jerry-uploader -- cat /uploads/sentinel
```
