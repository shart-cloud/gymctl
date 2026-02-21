#!/bin/bash
set -e

if grep -q 'kubectl drain' /tmp/jerry-upgrade.sh; then
  echo "Script includes 'kubectl drain'"
  exit 0
fi

echo "Script must include 'kubectl drain' before upgrading worker node packages"
echo "Drain the worker first to safely reschedule workloads before modifying the node"
exit 1
