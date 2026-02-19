If direct edit is blocked or not applied reliably, recreate the class safely:

```bash
kubectl delete storageclass jerry-retain
kubectl apply -f - <<'MANIFEST'
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: jerry-retain
provisioner: rancher.io/local-path
reclaimPolicy: Retain
volumeBindingMode: Immediate
MANIFEST
```

Then verify:

```bash
kubectl get storageclass jerry-retain -o jsonpath='{.reclaimPolicy}'; echo
```
