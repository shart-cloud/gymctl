Full recovery sequence:

```bash
# Find the released PV
PV=$(kubectl get pv -l jerry.gym/state=released -o jsonpath='{.items[0].metadata.name}')
echo "Recovering PV: $PV"

# Clear the stale claimRef
kubectl patch pv "$PV" --type=merge -p '{"spec":{"claimRef": null}}'

# Watch for Available phase
for i in $(seq 1 30); do
  phase=$(kubectl get pv "$PV" -o jsonpath='{.status.phase}')
  echo "PV phase: $phase"
  [ "$phase" = "Available" ] && break
  sleep 2
done

# Check if new claim bound (may take a few seconds)
kubectl get pvc jerry-new-claim
```

If the PVC is still Pending after the PV goes Available, delete and recreate it to
trigger a fresh bind attempt:

```bash
kubectl delete pvc jerry-new-claim
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: jerry-new-claim
spec:
  accessModes: ["ReadWriteOnce"]
  resources:
    requests:
      storage: 2Gi
  storageClassName: retained-local
EOF
```

Verify:

```bash
kubectl get pvc jerry-new-claim
kubectl get pv "$PV"
```
