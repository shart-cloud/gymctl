Validate snapshot and do restore dry-run (still on cp-1):

```bash
export KUBECONFIG=/etc/kubernetes/admin.conf
pod=$(kubectl -n kube-system get pod -l component=etcd -o jsonpath='{.items[0].metadata.name}')

kubectl -n kube-system exec "$pod" -- sh -lc '
  if command -v etcdutl >/dev/null 2>&1; then
    etcdutl snapshot status /var/lib/etcd/backups/jerry-etcd.db --write-out=table
    rm -rf /var/lib/etcd/restore-dryrun
    etcdutl snapshot restore /var/lib/etcd/backups/jerry-etcd.db --data-dir /var/lib/etcd/restore-dryrun
  else
    ETCDCTL_API=3 etcdctl snapshot status /var/lib/etcd/backups/jerry-etcd.db --write-out=table
    rm -rf /var/lib/etcd/restore-dryrun
    ETCDCTL_API=3 etcdctl snapshot restore /var/lib/etcd/backups/jerry-etcd.db --data-dir /var/lib/etcd/restore-dryrun
  fi
'
```

Then run:

```bash
gymctl check
```
