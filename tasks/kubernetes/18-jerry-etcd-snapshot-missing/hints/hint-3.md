Run a non-destructive restore drill to a separate data directory:

```bash
ETCD_POD=$(kubectl -n kube-system get pods -l component=etcd -o jsonpath='{.items[0].metadata.name}')
kubectl -n kube-system exec "$ETCD_POD" -- env ETCDCTL_API=3 etcdctl \
  snapshot restore /var/lib/etcd/etcd-snapshot.db \
  --data-dir /var/lib/etcd/restore-dryrun
```

Validate the restored output database:

```bash
kubectl -n kube-system exec "$ETCD_POD" -- env ETCDCTL_API=3 etcdctl \
  snapshot status /var/lib/etcd/restore-dryrun/member/snap/db -w table
```

Do not restore into `/var/lib/etcd` directly for this drill.
