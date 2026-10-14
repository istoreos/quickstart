#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
output_path="$(mktemp)"
trap 'rm -f "$output_path"' EXIT

set +e
ROUTER_ROLE=lan_gateway_candidate \
    PROTECT_CRITICAL_GATEWAY=1 \
    ALLOW_GATEWAY_LOAD_TEST=0 \
    "${SCRIPT_DIR}/lan-device-load-check.sh" >"$output_path" 2>&1
status=$?
set -e

[ "$status" -eq 3 ] || {
    echo "critical gateway load check must refuse with exit 3, got $status" >&2
    cat "$output_path" >&2
    exit 1
}
grep -q 'refusing load test on critical LAN gateway' "$output_path" || {
    echo "critical gateway refusal must explain the safety boundary" >&2
    cat "$output_path" >&2
    exit 1
}

echo "critical gateway load guard ok"
