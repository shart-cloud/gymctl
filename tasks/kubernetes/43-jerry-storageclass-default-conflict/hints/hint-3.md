Remove the default annotation from `fast-local`, leaving `standard-local` as the only default:

```bash
kubectl annotate storageclass fast-local \
  storageclass.kubernetes.io/is-default-class-

kubectl get storageclass
```

The PVC may need a nudge if it got stuck during the double-default state:

```bash
kubectl get pvc jerry-unqualified-pvc
# If still Pending, delete and recreate:
kubectl delete pvc jerry-unqualified-pvc
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: jerry-unqualified-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
EOF
```

Verify it binds to the default class:

```bash
kubectl get pvc jerry-unqualified-pvc
kubectl get pvc jerry-unqualified-pvc -o jsonpath='{.spec.storageClassName}'; echo
```
