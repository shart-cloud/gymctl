#!/bin/bash
set -e

if ! grep -q '\-\-sort-by' /tmp/jerry-jsonpath-broken.sh; then
  echo "QUERY4 section is missing --sort-by flag"
  echo "Add: --sort-by=.metadata.creationTimestamp"
  exit 1
fi

source /tmp/jerry-jsonpath-broken.sh 2>/dev/null || true

if [ -z "$QUERY4" ]; then
  echo "QUERY4 is empty even after adding --sort-by — check the jsonpath expression"
  exit 1
fi

echo "QUERY4 returned sorted pods: $QUERY4"
