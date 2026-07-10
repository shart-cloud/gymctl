# CKS-007 — Solution

The kube-apiserver is a **static pod**. You do not `kubectl edit` it — its knobs
are command-line flags in a manifest file on the control-plane node:
`/etc/kubernetes/manifests/kube-apiserver.yaml`. Harden the flags, keep the
cluster reachable.

## The fix (what you change)

Edit the manifest on node `cp-1` and make the container `command:` list contain:

| Flag | Value | Why |
| --- | --- | --- |
| `--anonymous-auth` | `false` | Stops unauthenticated callers reaching the API as `system:anonymous`. Off by policy; the API should require a real identity. |
| `--enable-admission-plugins` | `NodeRestriction,PodSecurity` | **NodeRestriction** limits what a kubelet can mutate (only its own Node and the Pods bound to it) — contains a stolen kubelet credential. **PodSecurity** turns on the built-in Pod Security Standards admission. Both, comma-separated, on one flag. |
| `--kubelet-certificate-authority` | `/etc/kubernetes/pki/ca.crt` | Makes the apiserver verify the kubelet's serving cert on the apiserver→kubelet channel (logs/exec/port-forward), closing a MITM. |
| `--tls-min-version` | `VersionTLS12` | Refuses TLS 1.0/1.1 downgrades. |
| `--insecure-port` | absent (or `0`) | The old unauthenticated localhost port. **Removed from kube-apiserver in k8s v1.20**, so on this cluster (v1.33) it simply does not exist — there is nothing to disable. Verify it is absent; do not add it (a nonzero value would make the apiserver refuse to start on modern versions). |

The real hardening deltas are the first four flags.

## The manual procedure (what a learner does on a real node)

```bash
# on the control-plane node (kind: `docker exec -it <node> bash`; real: ssh)
cp /etc/kubernetes/manifests/kube-apiserver.yaml \
   /etc/kubernetes/manifests/kube-apiserver.yaml.bak      # ALWAYS back up first
vi /etc/kubernetes/manifests/kube-apiserver.yaml
#   - set   - --anonymous-auth=false
#   - set   - --enable-admission-plugins=NodeRestriction,PodSecurity
#   - add   - --kubelet-certificate-authority=/etc/kubernetes/pki/ca.crt
#   - add   - --tls-min-version=VersionTLS12
# save. On a real kubeadm node the kubelet notices the change, restarts the
# apiserver, and you then poll for it to come back:
#   for i in $(seq 1 60); do kubectl get --raw=/livez && break; sleep 2; done
```

Keep every flag on its own `    - --flag=value` line inside the `command:`
sequence — the indentation must match its siblings or the kubelet will not parse
the pod.

`solve.sh` does exactly this from the host via `docker exec` (idempotent, leaves
`.bak`). The playtest applies the fix with `bash solution/solve.sh`.

## The traps

1. **Back up before you touch it.** A syntactically bad edit restarts the
   apiserver into a crash and can lock you out of the cluster. `.bak` beside the
   file is your undo. Worst case on the gym: `gymctl start` rebuilds the cluster.
2. **`--anonymous-auth=false` can crash the health probes on a real node.** The
   apiserver's liveness/readiness/startup probes GET `/livez` and `/readyz` over
   HTTPS **with no client cert** — i.e. anonymously. Those paths are in
   `--authorization-always-allow-paths` (authorization is waived) but that does
   **not** waive *authentication*: with anonymous auth off an unauthenticated
   probe can be rejected 401 and the kubelet will crash-loop the pod. On a real
   kubeadm node, if disabling anonymous auth breaks the probes, switch the probe
   scheme so it authenticates, or confirm your version exempts the health
   endpoints. **On this kind gym you will not observe a crash-loop — see below.**
3. **`--enable-admission-plugins` is one flag, not two.** `NodeRestriction` and
   `PodSecurity` go on the same flag, comma-separated. A second
   `--enable-admission-plugins` line silently wins/loses depending on order.

## Why kind does not hot-reload this edit (read this)

On a normal kubeadm node, saving the manifest makes the kubelet restart the
apiserver with the new flags. **On kind (single node, this gym) it does not** —
and that is not a bug in your edit. It is a structural chicken-and-egg:

- The kubelet's static-pod file source emits an "updated pod" event once, when
  the file content changes.
- Applying that update to the **kube-apiserver** means the kubelet must
  delete+recreate the apiserver's *mirror pod* through the API — but recreating
  the apiserver takes the API **down** for ~30s, so that mirror-pod update fails.
- kind runs the kubelet with `fileCheckFrequency: 0` (fsnotify only, no periodic
  rescan), so the file source does **not re-emit** the unchanged-since content,
  and the failed update is never retried. The apiserver stays frozen on its
  boot-time flags until the whole cluster is recreated.
- The **kube-scheduler / kube-controller-manager** edit *does* take effect on
  kind, because editing them does not take the API down — so the mirror-pod
  update succeeds. This asymmetry is specific to editing the component that
  *serves* the API.

Consequence for this exercise: the checks assert the **manifest content on the
node** (exactly what a CIS / kube-bench audit inspects), plus that the cluster
stayed reachable. There is no runtime "anonymous request is refused" check,
because the running apiserver never actually changes on kind — a runtime probe
would return the same result before and after your fix and prove nothing. On a
real kubeadm control-plane (or a VM-backed node) the identical edit restarts the
apiserver and the controls are enforced for real.

## Verify by hand

```bash
# from the host (cp-1 == jerry-gym-control-plane):
docker exec jerry-gym-control-plane grep -E -- \
  '--anonymous-auth=|--enable-admission-plugins=|--kubelet-certificate-authority=|--tls-min-version=' \
  /etc/kubernetes/manifests/kube-apiserver.yaml
kubectl get --raw=/livez           # ok
kubectl get nodes                  # cluster still reachable
```
