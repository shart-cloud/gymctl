This is a kubelet outage in the kind node container.

Find the node/container name and check kubelet service state:

```bash
NODE=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')
docker exec "$NODE" systemctl is-active kubelet
```

If it is inactive, start it.
