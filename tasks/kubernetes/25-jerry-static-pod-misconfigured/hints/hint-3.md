Fix the bad flag in `/etc/kubernetes/manifests/kube-scheduler.yaml` and save.
Kubelet will reconcile the static pod automatically.

Then verify:

```bash
kubectl -n kube-system get pods -l component=kube-scheduler -w
kubectl run sched-probe --image=registry.k8s.io/pause:3.10 --restart=Never
kubectl get pod sched-probe -o wide
```

If the probe pod gets a node assignment, scheduling is back.
