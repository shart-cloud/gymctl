#!/usr/bin/env bash

# gymctl shell integration. Sourcing this file is equivalent to adding
#   eval "$(gymctl shell-init)"
# to your ~/.bashrc or ~/.zshrc. The function itself lives in the binary so it
# always matches the installed gymctl version.

if command -v gymctl >/dev/null 2>&1; then
    eval "$(command gymctl shell-init)"
fi
