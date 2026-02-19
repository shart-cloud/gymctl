This is a static pod issue, so check the node-local manifest:

```bash
node=$(kubectl get nodes -o name | sed 's#node/##' | grep -E 'control-plane|master' | head -n1)
docker exec -it "$node" sh
ls -l /etc/kubernetes/manifests
cat /etc/kubernetes/manifests/kube-scheduler.yaml
```

Look for a broken scheduler `--kubeconfig` flag.
