#!/bin/bash
set -e

if grep -q '1\.30' /tmp/jerry-upgrade.sh; then
  echo "Script includes v1.30 as intermediate step"
  exit 0
fi

echo "Script must include v1.30 as an intermediate upgrade step"
echo "kubeadm only allows one minor version at a time: v1.29 -> v1.30 -> v1.31"
echo "You cannot jump directly from v1.29 to v1.31"
exit 1
