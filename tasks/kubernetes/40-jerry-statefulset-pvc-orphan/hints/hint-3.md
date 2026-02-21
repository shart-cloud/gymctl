Delete the wrong StatefulSet and its PVCs, then redeploy the original:

```bash
kubectl delete statefulset jerry-db-v2
kubectl delete pvc data-jerry-db-v2-0 data-jerry-db-v2-1

kubectl apply -f - <<'EOF'
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: jerry-db
spec:
  serviceName: jerry-db
  replicas: 2
  selector:
    matchLabels:
      app: jerry-db
  template:
    metadata:
      labels:
        app: jerry-db
    spec:
      containers:
      - name: db
        image: busybox:1.36
        command: ["sh", "-c", "sleep 3600"]
        volumeMounts:
        - name: data
          mountPath: /data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: fast-local
      resources:
        requests:
          storage: 1Gi
EOF

kubectl rollout status statefulset/jerry-db --timeout=120s
```

Verify the original data survived:

```bash
kubectl exec jerry-db-0 -- cat /data/sentinel
kubectl exec jerry-db-1 -- cat /data/sentinel
```
