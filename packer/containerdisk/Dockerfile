# KubeVirt containerDisk for the gymctl Kubernetes gym.
#
# A containerDisk is just an OCI image whose only job is to carry a bootable
# VM disk at /disk/. KubeVirt copies it out at VM start. We use scratch to keep
# the image to just the disk layer.
#
# Built by scripts/build-containerdisk.sh (emit-only; not pushed).
FROM scratch

# The baked Ubuntu 22.04 + Kubernetes qcow2. KubeVirt auto-detects the qcow2.
ADD disk.qcow2 /disk/disk.qcow2
