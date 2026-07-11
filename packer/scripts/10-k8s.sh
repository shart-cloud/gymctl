#!/usr/bin/env bash
#
# Bake Kubernetes into the image: containerd + kernel prereqs + kubeadm/kubelet/
# kubectl from pkgs.k8s.io, and pre-pull the control-plane images into containerd.
#
# Runs as root (via `sudo -E bash`) inside the Packer build VM.
# K8S_VERSION is the pkgs.k8s.io stable channel, e.g. "v1.36".
set -euxo pipefail

K8S_VERSION="${K8S_VERSION:-v1.36}"
export DEBIAN_FRONTEND=noninteractive

echo "==> Baking Kubernetes channel ${K8S_VERSION}"

# ---------------------------------------------------------------------------
# 1. Base packages + Docker apt repo (for containerd.io)
# ---------------------------------------------------------------------------
apt-get update
apt-get install -y --no-install-recommends \
  apt-transport-https ca-certificates curl gnupg lsb-release conntrack socat \
  qemu-guest-agent

# qemu-guest-agent lets the libvirt host read the VM's IP directly from inside
# the guest (vagrant-libvirt `qemu_use_agent = true`), instead of the flaky
# 2-second DHCP-lease poll that times out on WSL2. Enable it so it starts on
# boot (the virtio-serial channel is added by vagrant-libvirt).
systemctl enable qemu-guest-agent.service || true

install -m 0755 -d /etc/apt/keyrings

# Docker repo -> containerd.io
curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
  | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
  > /etc/apt/sources.list.d/docker.list

# Kubernetes repo (pkgs.k8s.io) for the requested stable channel.
curl -fsSL "https://pkgs.k8s.io/core:/stable:/${K8S_VERSION}/deb/Release.key" \
  | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
chmod a+r /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] \
https://pkgs.k8s.io/core:/stable:/${K8S_VERSION}/deb/ /" \
  > /etc/apt/sources.list.d/kubernetes.list

apt-get update

# ---------------------------------------------------------------------------
# 2. Kernel prereqs: modules + sysctls, swap off
# ---------------------------------------------------------------------------
cat > /etc/modules-load.d/k8s.conf <<'EOF'
overlay
br_netfilter
EOF
modprobe overlay
modprobe br_netfilter

cat > /etc/sysctl.d/99-kubernetes-cri.conf <<'EOF'
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sysctl --system

# Disable swap now and persistently.
swapoff -a || true
sed -ri 's/^([^#].*\sswap\s.*)$/#\1/' /etc/fstab || true

# ---------------------------------------------------------------------------
# 3. containerd, configured with SystemdCgroup = true
# ---------------------------------------------------------------------------
apt-get install -y --no-install-recommends containerd.io

mkdir -p /etc/containerd
containerd config default > /etc/containerd/config.toml
# Use the systemd cgroup driver (required to match kubelet).
sed -ri 's/(\s*SystemdCgroup\s*=\s*)false/\1true/' /etc/containerd/config.toml
grep -q 'SystemdCgroup = true' /etc/containerd/config.toml

systemctl enable containerd
systemctl restart containerd
# Give containerd a moment to bring up its socket.
for i in $(seq 1 30); do
  [ -S /run/containerd/containerd.sock ] && break
  sleep 1
done
containerd --version

# ---------------------------------------------------------------------------
# 4. kubeadm / kubelet / kubectl (held)
# ---------------------------------------------------------------------------
apt-get install -y --no-install-recommends kubelet kubeadm kubectl
apt-mark hold kubelet kubeadm kubectl
systemctl enable kubelet   # kubelet crash-loops until kubeadm init/join; expected.

echo "==> Baked versions:"
kubeadm version
kubectl version --client
kubelet --version

# ---------------------------------------------------------------------------
# 5. Pre-pull control-plane images into containerd (the big speedup)
# ---------------------------------------------------------------------------
echo "==> kubeadm config images list:"
kubeadm config images list
echo "==> Pulling control-plane images into containerd (k8s.io namespace)..."
kubeadm config images pull --cri-socket unix:///run/containerd/containerd.sock

echo "==> Images now baked into containerd (ctr -n k8s.io images ls):"
ctr -n k8s.io images ls | awk '{print $1}'

# crictl is handy for confirming; install a matching version if not present.
if ! command -v crictl >/dev/null 2>&1; then
  echo "crictl not present (kubeadm pulls via containerd directly) — skipping."
else
  crictl --runtime-endpoint unix:///run/containerd/containerd.sock images || true
fi

echo "==> Kubernetes bake complete for channel ${K8S_VERSION}."
