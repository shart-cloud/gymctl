Create the snapshot from the etcd static pod container.

On cp-1:

```bash
export KUBECONFIG=/etc/kubernetes/admin.conf
pod=$(kubectl -n kube-system get pod -l component=etcd -o jsonpath='{.items[0].metadata.name}')
kubectl -n kube-system exec "$pod" -- sh -lc '
  mkdir -p /var/lib/etcd/backups
  ETCDCTL_API=3 etcdctl \
    --endpoints=https://127.0.0.1:2379 \
    --cacert=/etc/kubernetes/pki/etcd/ca.crt \
    --cert=/etc/kubernetes/pki/etcd/server.crt \
    --key=/etc/kubernetes/pki/etcd/server.key \
    snapshot save /var/lib/etcd/backups/jerry-etcd.db
'
```
