#!/bin/bash
set -e

source /tmp/jerry-jsonpath-broken.sh 2>/dev/null || true

count=$(echo "$QUERY1" | wc -w | tr -d ' ')
if [ "$count" -ge 3 ]; then
  echo "QUERY1 returned $count pod names: $QUERY1"
  exit 0
else
  echo "QUERY1 should return 3 pod names, got $count word(s): '$QUERY1'"
  echo "Fix: use .items[*].metadata.name (not .items.metadata.name)"
  exit 1
fi
