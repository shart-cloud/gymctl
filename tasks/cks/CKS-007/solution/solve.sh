#!/usr/bin/env bash
#
# CKS-007 solution — harden the kube-apiserver static pod manifest on cp-1.
#
# This is the tier's analogue of "kubectl apply -f solution/": you cannot apply
# a static-pod manifest edit through the API, so the fix is a node-file edit.
# We drive it from the HOST via `docker exec` into the kind control-plane node,
# mirroring exactly how the harness resolves `node: cp-1`
# (internal/environment/kind_provider.go: cp-1 -> jerry-gym-control-plane).
#
# A learner on a real cluster would instead SSH to the control-plane node and
# edit /etc/kubernetes/manifests/kube-apiserver.yaml by hand (see README.md).
#
# Idempotent: safe to run repeatedly. Leaves a .bak beside the manifest.
set -euo pipefail

NODE="${CKS007_NODE:-jerry-gym-control-plane}"
M=/etc/kubernetes/manifests/kube-apiserver.yaml

if ! docker exec "$NODE" test -f "$M"; then
  echo "kube-apiserver manifest not found on $NODE at $M" >&2
  exit 1
fi

echo "Hardening $M on $NODE ..."

# Back up once (never clobber an existing .bak).
docker exec "$NODE" sh -c "[ -f ${M}.bak ] || cp ${M} ${M}.bak"

# 1. Disable anonymous auth. Add the flag if it is missing entirely, then
#    normalise any =true to =false.
docker exec "$NODE" grep -q -- '--anonymous-auth=' "$M" \
  || docker exec "$NODE" sed -i 's#^\(    - kube-apiserver\)$#\1\n    - --anonymous-auth=false#' "$M"
docker exec "$NODE" sed -i 's#--anonymous-auth=true#--anonymous-auth=false#' "$M"

# 2. Enable NodeRestriction AND PodSecurity (replace whatever list is there).
docker exec "$NODE" sed -i 's#--enable-admission-plugins=.*#--enable-admission-plugins=NodeRestriction,PodSecurity#' "$M"

# 3. Pin the kubelet serving CA (add the flag if absent).
docker exec "$NODE" grep -q -- '--kubelet-certificate-authority=' "$M" \
  || docker exec "$NODE" sed -i 's#^\(    - kube-apiserver\)$#\1\n    - --kubelet-certificate-authority=/etc/kubernetes/pki/ca.crt#' "$M"

# 4. Set the TLS floor to 1.2 (add the flag if absent).
docker exec "$NODE" grep -q -- '--tls-min-version=' "$M" \
  || docker exec "$NODE" sed -i 's#^\(    - kube-apiserver\)$#\1\n    - --tls-min-version=VersionTLS12#' "$M"

# 5. --insecure-port was removed in k8s v1.20 and is absent on kind v1.33+.
#    Nothing to add; if a stale =NNNN ever appears, force it to 0.
docker exec "$NODE" sed -i 's#--insecure-port=[0-9]\+#--insecure-port=0#' "$M"

echo "Done. Hardened flags now present in the manifest:"
docker exec "$NODE" grep -E -- '--anonymous-auth=|--enable-admission-plugins=|--kubelet-certificate-authority=|--tls-min-version=' "$M" || true

# NOTE: on kind the running apiserver does NOT pick this up (see README.md);
# the cluster stays healthy on its original args. On a real kubeadm node the
# kubelet restarts the apiserver here and you would poll for it to come back.
