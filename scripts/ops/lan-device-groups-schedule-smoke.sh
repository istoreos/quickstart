#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
MODE="${M59_SMOKE_MODE:-local}"
A_HOST="${A_HOST:-root@192.168.30.1}"
B_HOST="${B_HOST:-root@192.168.30.244}"
SID="${SID:-lan-device-m59-smoke}"

remote() { host="$1"; shift; ssh -o BatchMode=yes -o ConnectTimeout=8 "$host" "$@"; }
fingerprint() { remote "$1" 'sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall 2>/dev/null; pidof dnsmasq'; }

run_local() {
    cd "$PROJECT_ROOT/backend"
    go test ./service -run 'Test(DeviceGroupPolicyPrecedenceAndExplanation|DeviceGroupSchedule|DeviceScheduleReportsNextBoundary|SingleDeviceScheduleAndQuota|DeviceGroupReconcileUsesOneBatch|DefaultGroupBatch|CollectedGroupBatch|DeviceGroupRepeatedTick|DeviceGroupRestartAfterMissedWindow|DeviceGroupScheduleRestores|DeviceGroupGlobalPolicy)' -count=1
    cd "$PROJECT_ROOT"
    node --test web/test/deviceGroups.test.mjs
    printf 'SUMMARY result=pass mode=local precedence=global-group-device batch=atomic reloads=max-one schedule=enter-exit-cross-midnight\n'
}

run_device() {
    [ "${M59_DEVICE_SMOKE:-0}" = 1 ] || { echo 'device mode requires M59_DEVICE_SMOKE=1' >&2; exit 2; }
    command -v jq >/dev/null
    temp_dir="$(mktemp -d /tmp/quickstart-m59.XXXXXX)"
    trap 'rm -f "$temp_dir"/*; rmdir "$temp_dir" 2>/dev/null || true' EXIT HUP INT TERM
    for role_host in "A:$A_HOST" "B:$B_HOST"; do
        role="${role_host%%:*}"; host="${role_host#*:}"
        before="$(fingerprint "$host")"
        remote "$host" "curl -fsS --max-time 10 -H 'X-Forwarded-Sid: ${SID}' 'http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2/device-groups/'" >"$temp_dir/$role.json"
        jq -e '.result.version | type == "string" and length > 0' "$temp_dir/$role.json" >/dev/null
        jq -e '(.result.groups | type == "array") and (.result.effective | type == "array")' "$temp_dir/$role.json" >/dev/null
        [ "$before" = "$(fingerprint "$host")" ]
    done
    printf 'SUMMARY result=pass mode=device contracts=readable A=read-only B=read-only config_drift=none stateful-boundaries=isolated\n'
}

case "$MODE" in local) run_local ;; device) run_device ;; *) echo 'M59_SMOKE_MODE must be local or device' >&2; exit 2 ;; esac
