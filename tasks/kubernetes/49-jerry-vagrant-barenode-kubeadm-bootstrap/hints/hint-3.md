On worker, run the join command from control-plane output.

Install CNI from control-plane:

```bash
export KUBECONFIG=/etc/kubernetes/admin.conf
kubectl apply -f https://raw.githubusercontent.com/flannel-io/flannel/master/Documentation/kube-flannel.yml
kubectl get nodes
```

Optional host access after bootstrap:

```bash
eval "$(gymctl kubeconfig --format env)"
kubectl get nodes -o wide
```
