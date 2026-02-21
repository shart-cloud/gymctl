**Option A — Uncordon the target node (recommended for this lab):**

```bash
TARGET=$(kubectl get nodes -l jerry.gym/role=cordoned-target -o jsonpath='{.items[0].metadata.name}')
echo "Uncordoning: $TARGET"

kubectl uncordon "$TARGET"
kubectl get nodes

# The pod should schedule within seconds
kubectl get pod jerry-local-app -w
```

**Option B — Update PV nodeAffinity to the available node:**

```bash
AVAILABLE=$(kubectl get nodes -l jerry.gym/role=available-worker -o jsonpath='{.items[0].metadata.name}')

# Create storage directory on the available node
docker exec "$AVAILABLE" mkdir -p /mnt/local-storage/vol1

# Patch the PV nodeAffinity
kubectl patch pv jerry-local-pv --type=json -p "[
  {\"op\":\"replace\",
   \"path\":\"/spec/nodeAffinity/required/nodeSelectorTerms/0/matchExpressions/0/values/0\",
   \"value\":\"${AVAILABLE}\"}
]"

# Recreate the pod to trigger reschedule
kubectl delete pod jerry-local-app
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: jerry-local-app
  labels:
    app: jerry-local-app
spec:
  containers:
  - name: app
    image: busybox:1.36
    command: ["sh", "-c", "echo running && sleep 3600"]
    volumeMounts:
    - name: local-data
      mountPath: /data
  volumes:
  - name: local-data
    persistentVolumeClaim:
      claimName: jerry-local-claim
EOF

kubectl wait --for=condition=Ready pod/jerry-local-app --timeout=60s
```
