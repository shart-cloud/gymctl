Use the existing `fast-local` class:

```bash
kubectl delete pod jerry-storage-app
kubectl delete pvc jerry-data
kubectl apply -f - <<'MANIFEST'
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: jerry-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: fast-local
MANIFEST

kubectl apply -f - <<'MANIFEST'
apiVersion: v1
kind: Pod
metadata:
  name: jerry-storage-app
spec:
  containers:
    - name: app
      image: busybox:1.36
      command: ["sh", "-c", "echo starting && sleep 3600"]
      volumeMounts:
        - name: data
          mountPath: /data
  volumes:
    - name: data
      persistentVolumeClaim:
        claimName: jerry-data
MANIFEST
```

`spec.storageClassName` is immutable, so recreation is the safest fix path.
