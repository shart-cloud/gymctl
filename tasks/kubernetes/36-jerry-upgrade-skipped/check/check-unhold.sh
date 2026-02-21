#!/bin/bash
set -e

if grep -q 'apt-mark unhold' /tmp/jerry-upgrade.sh; then
  echo "Script includes 'apt-mark unhold'"
  exit 0
fi

echo "Script must include 'apt-mark unhold' before 'apt-get install'"
echo "kubeadm, kubelet, and kubectl are pinned (held) by default — they must be unheld before upgrading"
exit 1
