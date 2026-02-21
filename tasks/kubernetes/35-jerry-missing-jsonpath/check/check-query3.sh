#!/bin/bash
set -e

source /tmp/jerry-jsonpath-broken.sh 2>/dev/null || true

count=$(echo "$QUERY3" | wc -w | tr -d ' ')
if [ "$count" -ge 3 ]; then
  echo "QUERY3 returned $count image(s): $QUERY3"
  exit 0
else
  echo "QUERY3 should return at least 3 images, got $count word(s): '$QUERY3'"
  echo "Fix: images live at .items[*].spec.containers[*].image (not .items.containers.image)"
  exit 1
fi
