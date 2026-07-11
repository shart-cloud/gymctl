#!/usr/bin/env bash
#
# Blessed entrypoint for the gymctl image build.
#   1. downloads + checksum-verifies the Ubuntu 22.04 cloud image
#   2. packer init  (installs the qemu plugin)
#   3. packer build (KVM, headless) -> libvirt box + KubeVirt containerDisk
#
# Idempotent: re-runs reuse the cached base image and overwrite output.
#
# Usage:
#   ./build.sh                 # bake the default k8s channel (v1.36)
#   ./build.sh -var k8s_version=v1.34
set -euo pipefail
cd "$(dirname "$0")"

IMG_URL="https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
SUMS_URL="https://cloud-images.ubuntu.com/jammy/current/SHA256SUMS"
IMG="output/jammy-server-cloudimg-amd64.img"

mkdir -p output

echo "==> Fetching + verifying Ubuntu 22.04 cloud image"
curl -fsSL -o output/SHA256SUMS "$SUMS_URL"
WANT="$(grep 'jammy-server-cloudimg-amd64.img' output/SHA256SUMS | awk '{print $1}')"
if [ -f "$IMG" ] && echo "${WANT}  ${IMG}" | sha256sum -c - >/dev/null 2>&1; then
  echo "    cached base image OK ($WANT)"
else
  echo "    downloading base image..."
  curl -fSL -o "$IMG" "$IMG_URL"
  echo "${WANT}  ${IMG}" | sha256sum -c -
fi

# Vagrant's well-known insecure private key (public — it ships in every Vagrant
# install; `vagrant up` swaps in a unique key on first boot). Packer uses it to
# SSH into the build VM. Fetched here rather than committed, so the repo never
# carries a file containing a "PRIVATE KEY" block (keeps secret scanners happy).
KEY="vagrant_insecure_key"
if [ ! -f "$KEY" ]; then
  echo "==> Fetching Vagrant insecure private key"
  curl -fsSL -o "$KEY" "https://raw.githubusercontent.com/hashicorp/vagrant/main/keys/vagrant"
  chmod 600 "$KEY"
fi

echo "==> packer init"
packer init .

echo "==> packer build (use -force to overwrite prior output)"
rm -rf output/qemu
packer build -force "$@" .

echo "==> Done. Artifacts under packer/output/"
ls -lh output/*.box 2>/dev/null || true
