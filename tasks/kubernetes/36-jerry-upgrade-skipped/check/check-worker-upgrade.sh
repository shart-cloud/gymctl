#!/bin/bash
set -e

if grep -q 'kubeadm upgrade node' /tmp/jerry-upgrade.sh; then
  echo "Script includes 'kubeadm upgrade node'"
  exit 0
fi

echo "Script must include 'kubeadm upgrade node' for worker node upgrades"
echo "'kubeadm upgrade node' syncs the kubelet configuration on each worker after the binary is upgraded"
exit 1
