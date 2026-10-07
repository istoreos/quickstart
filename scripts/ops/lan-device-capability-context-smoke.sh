#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
MODE="${M56_SMOKE_MODE:-local}"
A_HOST="${A_HOST:-root@192.168.30.1}"
B_HOST="${B_HOST:-root@192.168.30.244}"
SID="${SID:-lan-device-m56-smoke}"

remote() {
    m56_host="$1"
    shift
    ssh -o BatchMode=yes -o ConnectTimeout=8 "$m56_host" "$@"
}

read_api() {
    m56_host="$1"
    m56_path="$2"
    remote "$m56_host" "curl -fsS --max-time 10 -H 'X-Forwarded-Sid: ${SID}' 'http://127.0.0.1:3038${m56_path}'"
}

config_fingerprint() {
    remote "$1" 'sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall; pidof dnsmasq'
}

run_local() {
    cd "$PROJECT_ROOT/backend"
    go test ./service -run 'Test(Capability|SystemCapability|RouterContext|BuildDeviceManagementCapabilit|LanGlobalConfigServiceReportsOptional|AccessPolicyDoesNotDepend|FirewallCapability)' -count=1
    cd "$PROJECT_ROOT"
    node --test web/test/formalDeviceUi.test.mjs
    printf 'SUMMARY result=pass mode=local authority=covered capability_degradation=isolated draft=preserved native_service=verified A=not-connected\n'
}

run_device() {
    [ "${M56_DEVICE_SMOKE:-0}" = 1 ] || {
        echo "device mode requires M56_DEVICE_SMOKE=1" >&2
        exit 2
    }
    command -v jq >/dev/null
    temp_dir="$(mktemp -d /tmp/quickstart-m56.XXXXXX)"
    cleanup() {
        rm -f "$temp_dir/a-context.json" "$temp_dir/b-context.json" "$temp_dir/b-capabilities.json"
        rmdir "$temp_dir" 2>/dev/null || true
    }
    trap cleanup EXIT HUP INT TERM

    a_before="$(config_fingerprint "$A_HOST")"
    b_before="$(config_fingerprint "$B_HOST")"
    read_api "$A_HOST" '/cgi-bin/luci/istore/lanctrl/v2/router-context/' >"$temp_dir/a-context.json"
    read_api "$B_HOST" '/cgi-bin/luci/istore/lanctrl/v2/router-context/' >"$temp_dir/b-context.json"
    read_api "$B_HOST" '/cgi-bin/luci/istore/lanctrl/globalConfigs/' >"$temp_dir/b-capabilities.json"

    jq -e '.result.localAddress == "192.168.30.1" and .result.dhcpAuthority == "local" and .result.routeEditability.editable == true' "$temp_dir/a-context.json" >/dev/null
    jq -e '.result.localAddress == "192.168.30.244" and .result.topologyPosition == "downstream_router" and .result.routeEditability.editable == false and (.result.dhcpAuthority == "external_observed" or .result.dhcpAuthority == "none_detected")' "$temp_dir/b-context.json" >/dev/null
    jq -e '
      .result.capabilities.items.internet_access.state == "available" and
      .result.capabilities.items.device_speed_limit.state == "available" and
      .result.capabilities.items.floating_gateway.state == "available"
    ' "$temp_dir/b-capabilities.json" >/dev/null
    remote "$B_HOST" '/etc/init.d/quickstart-netpolicy running; test -z "$(pidof bandix 2>/dev/null || true)"'

    a_after="$(config_fingerprint "$A_HOST")"
    b_after="$(config_fingerprint "$B_HOST")"
    [ "$a_before" = "$a_after" ]
    [ "$b_before" = "$b_after" ]
    printf 'SUMMARY result=pass mode=device A=local-editable-read-only B=downstream-read-only capabilities=available native_service=running config_drift=none\n'
}

case "$MODE" in
    local) run_local ;;
    device) run_device ;;
    *) echo "M56_SMOKE_MODE must be local or device" >&2; exit 2 ;;
esac
