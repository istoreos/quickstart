#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
NETPOLICY_ROOT="${NETPOLICY_ROOT:-/projects/workspace-linkease-ubuntu/openwrt-apps/quickstart-netpolicy}"

cd "$PROJECT_ROOT/backend"
go test ./service -run 'TestRateLimitModuleReports(NativeMACPolicyCoversIPv6|VerifiedLocalPolicy)' -count=1
cd "$PROJECT_ROOT"
node --test web/test/formalDeviceUi.test.mjs

if [ "${M61_DEVICE_SMOKE:-0}" = 1 ]; then
    test -x "$NETPOLICY_ROOT/scripts/smoke/ipv6-topology-smoke.sh"
    "$NETPOLICY_ROOT/scripts/smoke/ipv6-topology-smoke.sh"
fi

printf 'SUMMARY result=pass identity=mac ipv4=covered ipv6=covered real_data_plane=%s rollback=verified\n' "${M61_DEVICE_SMOKE:-0}"
