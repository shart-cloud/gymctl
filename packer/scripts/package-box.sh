#!/usr/bin/env bash
#
# Package the baked qcow2 into a vagrant-libvirt box.
#
# Packer's built-in `vagrant` post-processor has no libvirt provider, so we
# build the box by hand: a gzip'd tar of
#   box.img         (the qcow2, must be named exactly "box.img")
#   metadata.json   {"provider":"libvirt","format":"qcow2","virtual_size":<GB>}
#   Vagrantfile     (minimal, sets the libvirt driver)
#
# Runs on the HOST (shell-local post-processor) from the packer/ directory.
set -euxo pipefail

BOX_NAME="${BOX_NAME:?BOX_NAME not set}"
QCOW2="output/qemu/box.img"
STAGE="output/box-stage"
OUT="output/${BOX_NAME}"

[ -f "$QCOW2" ] || { echo "ERROR: $QCOW2 not found"; exit 1; }

rm -rf "$STAGE"
mkdir -p "$STAGE"

cp "$QCOW2" "$STAGE/box.img"

# virtual_size in whole GB (rounded up) from the qcow2's virtual size.
# Parse the parenthesized byte count from plain `qemu-img info` (the JSON output
# contains more than one virtual-size field, so we avoid it).
VSIZE_BYTES=$(qemu-img info "$QCOW2" | sed -n 's/.*virtual size:.*(\([0-9]\+\) bytes).*/\1/p' | head -1)
VSIZE_GB=$(( (VSIZE_BYTES + 1073741823) / 1073741824 ))

cat > "$STAGE/metadata.json" <<EOF
{
  "provider": "libvirt",
  "format": "qcow2",
  "virtual_size": ${VSIZE_GB}
}
EOF

cat > "$STAGE/Vagrantfile" <<'EOF'
# Minimal Vagrantfile baked into the gymctl libvirt box.
Vagrant.configure("2") do |config|
  config.vm.provider :libvirt do |libvirt|
    libvirt.driver = "kvm"
  end
  # The vagrant user + insecure key are baked in; keep vagrant's default
  # insecure-key replacement behaviour on first `vagrant up`.
end
EOF

rm -f "$OUT"
# Order matters a little for readers; keep box.img last (largest).
tar -czf "$OUT" -C "$STAGE" metadata.json Vagrantfile box.img

echo "==> Wrote libvirt box: $OUT"
ls -lh "$OUT"
echo "==> Box contents:"
tar -tzvf "$OUT"
rm -rf "$STAGE"
