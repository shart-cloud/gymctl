#!/usr/bin/env bash
#
# De-provision the image so it boots generic on any host and compresses small.
# Runs as root inside the build VM, just before shutdown.
set -euxo pipefail

export DEBIAN_FRONTEND=noninteractive

echo "==> apt cleanup"
apt-get -y autoremove || true
apt-get -y clean
rm -rf /var/lib/apt/lists/*

echo "==> reset cloud-init so it re-runs on the next (real) boot"
cloud-init clean --logs || true
rm -rf /var/lib/cloud/instances/* /var/lib/cloud/instance || true

echo "==> truncate machine-id (regenerated on next boot)"
truncate -s 0 /etc/machine-id
rm -f /var/lib/dbus/machine-id || true
ln -sf /etc/machine-id /var/lib/dbus/machine-id || true

echo "==> clear logs, tmp, shell history, ssh host keys"
find /var/log -type f -exec truncate -s 0 {} \; || true
rm -rf /tmp/* /var/tmp/* || true
rm -f /home/vagrant/.bash_history /root/.bash_history || true
# Regenerate SSH host keys on first real boot.
rm -f /etc/ssh/ssh_host_* || true

echo "==> zero-fill free space so the qcow2 compresses (then trim)"
dd if=/dev/zero of=/EMPTY bs=16M 2>/dev/null || true
rm -f /EMPTY
sync
fstrim -av || true

echo "==> cleanup complete"
