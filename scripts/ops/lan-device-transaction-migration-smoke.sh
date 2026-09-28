#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
MODE="${M58_SMOKE_MODE:-local}"
A_HOST="${A_HOST:-root@192.168.30.1}"
B_HOST="${B_HOST:-root@192.168.30.244}"
SID="${SID:-lan-device-m58-smoke}"

remote() { host="$1"; shift; ssh -o BatchMode=yes -o ConnectTimeout=8 "$host" "$@"; }
fingerprint() { remote "$1" 'sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall 2>/dev/null; pidof dnsmasq'; }

run_local() {
    cd "$PROJECT_ROOT/backend"
    go test ./service -run 'Test(LanDeviceMigration|TaskTransaction|DeviceProfile|DeviceNetworkPolicy|DevicePolicy)' -count=1
    cd "$PROJECT_ROOT"
    LAN_LAB_MODE=self-test ./scripts/ops/lan-device-isolation-lab.sh
    node --test web/test/formalDeviceUi.test.mjs
    printf 'SUMMARY result=pass mode=local migration=versioned-idempotent conflict=zero-write transaction=three-tasks rollback=distinguished recovery=distinguished A=not-connected\n'
}

run_device() {
    [ "${M58_DEVICE_SMOKE:-0}" = 1 ] || { echo 'device mode requires M58_DEVICE_SMOKE=1' >&2; exit 2; }
    command -v jq >/dev/null
    temp_dir="$(mktemp -d /tmp/quickstart-m58.XXXXXX)"
    trap 'rm -f "$temp_dir"/*; rmdir "$temp_dir" 2>/dev/null || true' EXIT HUP INT TERM
    for role_host in "A:$A_HOST" "B:$B_HOST"; do
        role="${role_host%%:*}"; host="${role_host#*:}"
        before="$(fingerprint "$host")"
        remote "$host" "curl -fsS --max-time 10 -H 'X-Forwarded-Sid: ${SID}' 'http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2/migration/plan/'" >"$temp_dir/$role-1.json"
        remote "$host" "curl -fsS --max-time 10 -H 'X-Forwarded-Sid: ${SID}' 'http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2/migration/plan/'" >"$temp_dir/$role-2.json"
        jq -e '.result.plan.version | type == "string" and length > 0' "$temp_dir/$role-1.json" >/dev/null
        [ "$(jq -r '.result.plan.version' "$temp_dir/$role-1.json")" = "$(jq -r '.result.plan.version' "$temp_dir/$role-2.json")" ]
        [ "$before" = "$(fingerprint "$host")" ]
    done
    printf 'SUMMARY result=pass mode=device migration_plan=stable-repeat A=read-only B=read-only config_drift=none apply=isolated-only\n'
}

case "$MODE" in local) run_local ;; device) run_device ;; *) echo 'M58_SMOKE_MODE must be local or device' >&2; exit 2 ;; esac
