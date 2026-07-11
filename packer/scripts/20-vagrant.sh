#!/usr/bin/env bash
#
# Make the baked cloud image usable as a vagrant-libvirt box. A vagrant-libvirt
# VM has NO cloud-init datasource at runtime, so cloud-init self-disables
# ("disabled-by-generator"). Anything that normally relies on cloud-init at boot
# must be made static here, or it silently never happens.
#
# Runs as root inside the Packer build VM.
set -euxo pipefail

# ---------------------------------------------------------------------------
# 1. MAC/NIC-agnostic DHCP netplan (LOWEST priority).
#    The build-time cloud-init netplan is frozen to the *build* NIC (e.g. ens3);
#    at runtime the NICs differ (ens6/ens7) and would stay DOWN with no IP. Match
#    any en*/eth* interface and DHCP it.
#
#    CRITICAL: vagrant-libvirt writes its OWN per-interface netplan giving the
#    private-network NIC (e.g. ens7) a static IP (e.g. 192.168.56.10), which
#    kubeadm advertises. netplan renders each ethernet to
#    /run/systemd/network/10-netplan-<KEY>.network and systemd-networkd applies
#    the FIRST match by lexical filename order. A catch-all keyed "all-en" sorts
#    BEFORE "ens7" and steals the private NIC with DHCP, dropping the static IP
#    (kubeadm then advertises an address no interface holds -> init hangs). So
#    key the catch-all "zz-*" to sort AFTER any interface-name-keyed file, so the
#    specific static config wins for its NIC and only leftover NICs get DHCP.
# ---------------------------------------------------------------------------
cat > /etc/netplan/50-cloud-init.yaml <<'EOF'
network:
  version: 2
  ethernets:
    zz-all-en:
      match:
        name: "en*"
      dhcp4: true
    zz-all-eth:
      match:
        name: "eth*"
      dhcp4: true
EOF
chmod 0600 /etc/netplan/50-cloud-init.yaml

# 2. Keep cloud-init from re-managing the network on any boot it does find a
#    datasource (it would re-freeze a NIC-specific config).
cat > /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg <<'EOF'
network: {config: disabled}
EOF

# ---------------------------------------------------------------------------
# 3. Regenerate SSH host keys on boot if missing.
#    Cleanup deletes /etc/ssh/ssh_host_* (box hygiene) expecting cloud-init to
#    regenerate them — which never runs here, so sshd fails to start and vagrant
#    sees "connection refused". This oneshot runs before sshd and recreates the
#    keys whenever they are absent.
# ---------------------------------------------------------------------------
cat > /etc/systemd/system/regen-ssh-host-keys.service <<'EOF'
[Unit]
Description=Regenerate SSH host keys if missing
Before=ssh.service
ConditionPathExists=!/etc/ssh/ssh_host_rsa_key

[Service]
Type=oneshot
ExecStart=/usr/bin/ssh-keygen -A

[Install]
WantedBy=multi-user.target
EOF
systemctl enable regen-ssh-host-keys.service

echo "==> vagrant-ification complete"
