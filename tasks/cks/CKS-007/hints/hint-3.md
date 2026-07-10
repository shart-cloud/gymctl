# CKS-007 — Hint 3 (cost 50)

On `cp-1`, back up and edit the manifest so the `command:` list looks like this
(only the relevant lines shown):

```yaml
spec:
  containers:
  - command:
    - kube-apiserver
    - --anonymous-auth=false
    - --enable-admission-plugins=NodeRestriction,PodSecurity
    - --kubelet-certificate-authority=/etc/kubernetes/pki/ca.crt
    - --tls-min-version=VersionTLS12
    # ...all the original flags stay...
    # NO --insecure-port line (removed in k8s v1.20)
```

Do it from the host (this is what `solution/solve.sh` runs):

```bash
NODE=jerry-gym-control-plane
M=/etc/kubernetes/manifests/kube-apiserver.yaml

docker exec "$NODE" sh -c "[ -f ${M}.bak ] || cp ${M} ${M}.bak"

# anonymous-auth -> false (add if absent, then normalise true->false)
docker exec "$NODE" grep -q -- '--anonymous-auth=' "$M" \
  || docker exec "$NODE" sed -i 's#^\(    - kube-apiserver\)$#\1\n    - --anonymous-auth=false#' "$M"
docker exec "$NODE" sed -i 's#--anonymous-auth=true#--anonymous-auth=false#' "$M"

# both admission plugins on one flag
docker exec "$NODE" sed -i 's#--enable-admission-plugins=.*#--enable-admission-plugins=NodeRestriction,PodSecurity#' "$M"

# kubelet CA + TLS floor (add if absent)
docker exec "$NODE" grep -q -- '--kubelet-certificate-authority=' "$M" \
  || docker exec "$NODE" sed -i 's#^\(    - kube-apiserver\)$#\1\n    - --kubelet-certificate-authority=/etc/kubernetes/pki/ca.crt#' "$M"
docker exec "$NODE" grep -q -- '--tls-min-version=' "$M" \
  || docker exec "$NODE" sed -i 's#^\(    - kube-apiserver\)$#\1\n    - --tls-min-version=VersionTLS12#' "$M"
```

Then confirm the cluster is still reachable:

```bash
kubectl get --raw=/livez     # ok
kubectl get nodes            # still works
```

The full, idempotent script is `solution/solve.sh`, and `solution/README.md`
explains why each flag matters and why kind does not hot-reload this edit.
