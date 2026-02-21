#!/bin/bash
set -e

if ! grep -q 'kubeadm upgrade plan' /tmp/jerry-upgrade.sh; then
  echo "Script must include 'kubeadm upgrade plan'"
  echo "Always run 'kubeadm upgrade plan' before 'kubeadm upgrade apply' to validate the upgrade"
  exit 1
fi

if ! grep -q 'kubeadm upgrade apply' /tmp/jerry-upgrade.sh; then
  echo "Script must include 'kubeadm upgrade apply'"
  exit 1
fi

plan_line=$(grep -n 'kubeadm upgrade plan' /tmp/jerry-upgrade.sh | head -1 | cut -d: -f1)
apply_line=$(grep -n 'kubeadm upgrade apply' /tmp/jerry-upgrade.sh | head -1 | cut -d: -f1)

if [ "$plan_line" -lt "$apply_line" ]; then
  echo "'kubeadm upgrade plan' (line $plan_line) appears before 'kubeadm upgrade apply' (line $apply_line)"
  exit 0
fi

echo "'kubeadm upgrade plan' must appear before 'kubeadm upgrade apply'"
echo "plan line: $plan_line, apply line: $apply_line"
exit 1
