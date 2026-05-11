#!/usr/bin/env bash

# gymctl shell wrapper for auto-cd on `gymctl start`.

gymctl() {
    if [[ "$1" == "start" ]]; then
        local output rc cd_target line
        output=$(command gymctl start --emit-cd "${@:2}" 2>&1)
        rc=$?

        while IFS= read -r line; do
            if [[ "$line" == __gymctl_cd:* ]]; then
                cd_target=${line#__gymctl_cd:}
                continue
            fi
            printf '%s\n' "$line"
        done <<< "$output"

        if [[ -n "$cd_target" && -d "$cd_target" ]]; then
            cd "$cd_target" || return $rc
        fi

        return $rc
    fi

    command gymctl "$@"
}
