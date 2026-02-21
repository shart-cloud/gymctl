**Option A: Restart the pod (quick fix)**

```bash
kubectl delete pod jerry-config-app

kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: jerry-config-app
  labels:
    app: jerry-config-app
spec:
  containers:
  - name: app
    image: busybox:1.36
    command: ["sh", "-c", "sleep 3600"]
    volumeMounts:
    - name: config
      mountPath: /etc/app/db-host
      subPath: db-host
  volumes:
  - name: config
    configMap:
      name: jerry-app-config
EOF

kubectl wait --for=condition=Ready pod/jerry-config-app --timeout=60s
kubectl exec jerry-config-app -- cat /etc/app/db-host
```

**Option B: Remove subPath (enables live updates)**

```bash
kubectl delete pod jerry-config-app

kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: jerry-config-app
  labels:
    app: jerry-config-app
spec:
  containers:
  - name: app
    image: busybox:1.36
    command: ["sh", "-c", "sleep 3600"]
    volumeMounts:
    - name: config
      mountPath: /etc/app
  volumes:
  - name: config
    configMap:
      name: jerry-app-config
EOF

kubectl wait --for=condition=Ready pod/jerry-config-app --timeout=60s
kubectl exec jerry-config-app -- cat /etc/app/db-host
```

Both options make the check pass. Option B is the production-correct approach.
