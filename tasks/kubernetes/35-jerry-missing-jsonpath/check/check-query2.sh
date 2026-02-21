#!/bin/bash
set -e

source /tmp/jerry-jsonpath-broken.sh 2>/dev/null || true

if [ -z "$QUERY2" ]; then
  echo "QUERY2 is empty — field path is wrong or field name case is incorrect"
  echo "Fix: jsonpath is case-sensitive. Use .spec.nodeName (capital N), not .spec.nodename"
  exit 1
fi

kubectl get node "$QUERY2" >/dev/null 2>&1 || {
  echo "QUERY2 value '$QUERY2' is not a valid node name in the cluster"
  exit 1
}

echo "QUERY2 returned valid node: $QUERY2"
