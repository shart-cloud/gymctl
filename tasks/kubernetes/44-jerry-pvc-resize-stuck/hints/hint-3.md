Full fix sequence:

```bash
# Step 1: Enable expansion on the StorageClass
kubectl patch storageclass expandable-local --type=merge \
  -p '{"allowVolumeExpansion": true}'

# Step 2: Resize the PVC
kubectl patch pvc jerry-app-data --type=merge \
  -p '{"spec":{"resources":{"requests":{"storage":"5Gi"}}}}'

# Step 3: Check the resize status
kubectl get pvc jerry-app-data
kubectl describe pvc jerry-app-data | grep -E "Capacity|Conditions|Resize|storage"
```

With local-path provisioner, the PVC spec updates immediately. The filesystem inside
the pod may not resize without a pod restart:

```bash
kubectl delete pod jerry-app
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: jerry-app
  labels:
    app: jerry-app
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
      claimName: jerry-app-data
EOF

kubectl wait --for=condition=Ready pod/jerry-app --timeout=60s
kubectl exec jerry-app -- df -h /data
```
