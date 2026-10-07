#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
MODE="${M55_SMOKE_MODE:-local}"
A_HOST="${A_HOST:-root@192.168.30.1}"
B_HOST="${B_HOST:-root@192.168.30.244}"
SID="${SID:-lan-device-m55-smoke}"

run_local() {
    cd "$PROJECT_ROOT/backend"
    go test ./service -run 'Test(DeviceInventoryManualDeviceIsPersistentNeverSeenAndMerges|DeviceInventoryKeepsBootHistoryAndStableIdentity|DeviceInventoryDoesNotTransferIdentityWhenIPIsReused|DeviceProfileTransactionIsIdempotentAndKeepsUnicodeOutOfDHCP|NormalizeStaticAssignmentInputAcceptsPortableHostname|NormalizeStaticAssignmentInputRejectsUnsafeHostnames|LanDHCPSettingsPlanRejectsUnsafePoolWithoutWriting|LanDHCPSettingsDisableAndRestoreInOneTaskModule)$' -count=1
    cd "$PROJECT_ROOT"
    node --test web/test/formalDeviceUi.test.mjs
    printf 'SUMMARY result=pass mode=local identity=merged addresses=stable hostname=separated dhcp=restored A=not-connected\n'
}

remote() {
    m55_host="$1"
    shift
    ssh -o BatchMode=yes -o ConnectTimeout=8 "$m55_host" "$@"
}

run_device() {
    [ "${M55_DEVICE_SMOKE:-0}" = 1 ] || {
        echo "device mode requires M55_DEVICE_SMOKE=1" >&2
        exit 2
    }
    command -v jq >/dev/null
    temp_dir="$(mktemp -d /tmp/quickstart-m55.XXXXXX)"
    remote_fixture="/tmp/quickstart-m55-dnsmasq-$$.conf"
    cleanup() {
        remote "$B_HOST" "rm -f '$remote_fixture'" >/dev/null 2>&1 || true
        rm -f "$temp_dir/devices.json"
        rmdir "$temp_dir" 2>/dev/null || true
    }
    trap cleanup EXIT HUP INT TERM

    a_before="$(remote "$A_HOST" 'sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall; pidof dnsmasq')"
    remote "$B_HOST" "curl -fsS --max-time 10 -H 'X-Forwarded-Sid: ${SID}' 'http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2/devices/'" >"$temp_dir/devices.json"
    jq -e '
      (.result.devices | length > 0) and
      (([.result.devices[].deviceId] | length) == ([.result.devices[].deviceId] | unique | length)) and
      all(.result.devices[];
        ([.addresses.current[]?.address] as $current |
         all(.addresses.historical[]?.address; ($current | index(.) | not))))
    ' "$temp_dir/devices.json" >/dev/null

    remote "$B_HOST" "set -eu; printf '%s\n' 'port=0' 'interface=lo' 'bind-interfaces' 'dhcp-range=192.0.2.100,192.0.2.110,255.255.255.0,1h' 'dhcp-host=02:00:00:00:55:01,m55-host,192.0.2.101' >'$remote_fixture'; dnsmasq --test -C '$remote_fixture' >/dev/null 2>&1; rm -f '$remote_fixture'"
    a_after="$(remote "$A_HOST" 'sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall; pidof dnsmasq')"
    [ "$a_before" = "$a_after" ]
    printf 'SUMMARY result=pass mode=device B_inventory=consistent staged_dnsmasq=healthy A=read-only cleanup=armed\n'
}

case "$MODE" in
    local) run_local ;;
    device) run_device ;;
    *) echo "M55_SMOKE_MODE must be local or device" >&2; exit 2 ;;
esac
