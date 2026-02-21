# Hint 3: Worker Node Steps

After the control plane is upgraded, each worker node needs its own sequence.
Before touching packages on a worker, drain it first:

```bash
kubectl drain <worker-node> --ignore-daemonsets --delete-emptydir-data
```

Then on each worker:

```bash
apt-mark unhold kubeadm && apt-get install -y kubeadm=<target>
kubeadm upgrade node
apt-mark unhold kubelet kubectl && apt-get install -y kubelet=<target> kubectl=<target>
kubectl uncordon <worker-node>
```

Add `kubectl drain`, `kubeadm upgrade node`, and the v1.30 intermediate steps to
`/tmp/jerry-upgrade.sh`.
