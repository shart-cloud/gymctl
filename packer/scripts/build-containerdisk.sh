#!/usr/bin/env bash
#
# Build a KubeVirt containerDisk OCI image from the baked qcow2.
# Emit-only: built locally, NOT pushed.
#
# Runs on the HOST (shell-local post-processor) from the packer/ directory.
set -euxo pipefail

CONTAINERDISK_IMAGE="${CONTAINERDISK_IMAGE:?CONTAINERDISK_IMAGE not set}"
QCOW2="output/qemu/box.img"
CTX="output/containerdisk-ctx"

[ -f "$QCOW2" ] || { echo "ERROR: $QCOW2 not found"; exit 1; }

rm -rf "$CTX"
mkdir -p "$CTX"
# KubeVirt containerDisk expects the disk under /disk inside the image.
cp "$QCOW2" "$CTX/disk.qcow2"
cp containerdisk/Dockerfile "$CTX/Dockerfile"

docker build -t "$CONTAINERDISK_IMAGE" "$CTX"

echo "==> Built containerDisk image: $CONTAINERDISK_IMAGE"
docker image ls "$CONTAINERDISK_IMAGE"
rm -rf "$CTX"
