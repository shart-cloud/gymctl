# CKS-007 — Hint 1 (free)

The kube-apiserver is a **static pod**, not an API object. You cannot
`kubectl edit` it — its configuration is command-line flags in a file **on the
control-plane node**:

```bash
# cp-1 is the kind control-plane container
docker exec -it jerry-gym-control-plane bash
cat /etc/kubernetes/manifests/kube-apiserver.yaml     # look at the command: list
```

Read the current flags. You are being asked to harden five things:

1. `--anonymous-auth` — it's `true`; it should be `false`.
2. `--enable-admission-plugins` — it lost `NodeRestriction` and never had
   `PodSecurity`; it should list **both**.
3. `--kubelet-certificate-authority` — missing; point it at the cluster CA.
4. `--tls-min-version` — missing; set a floor of TLS 1.2.
5. `--insecure-port` — should be absent (it was removed from modern
   kube-apiserver, so there is nothing to add here).

**Back the file up before you edit it.** A bad edit restarts the apiserver and
can lock you out. Read `solution/README.md` for how kind handles (and does not
hot-reload) this edit.

Docs: <https://kubernetes.io/docs/tasks/administer-cluster/securing-a-cluster/>
