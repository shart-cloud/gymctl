# Hint 2: Static Pod Manifests Live on the Node

The scheduler runs as a static pod — kubelet reads its manifest from
`/etc/kubernetes/manifests/` on the control-plane node. If the file is missing,
the pod simply never appears (no CrashLoop, no error — just silence).

Check whether the manifest is still there:

```bash
node=$(kubectl get nodes -l node-role.kubernetes.io/control-plane \
  -o jsonpath='{.items[0].metadata.name}')
docker exec "$node" ls /etc/kubernetes/manifests/
```

Is `kube-scheduler.yaml` listed?
