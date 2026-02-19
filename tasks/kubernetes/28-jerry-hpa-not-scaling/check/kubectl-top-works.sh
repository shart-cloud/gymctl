#!/bin/bash
set -e

# Try kubectl top nodes and check if it returns actual data
OUTPUT=$(kubectl top nodes 2>/dev/null || echo "ERROR")

if [[ "$OUTPUT" == "ERROR" ]]; then
    echo "✗ kubectl top nodes failed"
    exit 1
fi

# Check if output contains actual CPU/memory data (not just headers)
if echo "$OUTPUT" | grep -q "CPU(cores)" && echo "$OUTPUT" | grep -E "[0-9]+m.*[0-9]+Mi"; then
    echo "✓ kubectl top nodes returns metric data"
    exit 0
else
    echo "✗ kubectl top nodes doesn't return metric data"
    echo "Output: $OUTPUT"
    exit 1
fi