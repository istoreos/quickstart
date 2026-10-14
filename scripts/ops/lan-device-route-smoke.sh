#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
MODE="${M57_SMOKE_MODE:-local}"
A_HOST="${A_HOST:-root@192.168.30.1}"
B_HOST="${B_HOST:-root@192.168.30.244}"
C_HOST="${C_HOST:-root@192.168.30.93}"
D_HOST="${D_HOST:-root@192.168.30.7}"
SID="${SID:-lan-device-m57-smoke}"

remote() {
    m57_host="$1"
    shift
    ssh -o BatchMode=yes -o ConnectTimeout=8 "$m57_host" "$@"
}

fingerprint() {
    remote "$1" 'sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall; pidof dnsmasq'
}

run_local() {
    cd "$PROJECT_ROOT/backend"
    go test ./service -run 'Test(DeviceGroupPolicyPrecedenceAndExplanation|GatewayPolicy|ParseFailedIPv4Neighbors|MutateGatewayPolicy|DeviceNetworkPolicy|SystemNetworkRulesExplainsUnreachableGatewayAndRecovery|ClearNetworkRuleRoute|NetworkRulesBatchFailure)' -count=1
    cd "$PROJECT_ROOT"
    node --test web/test/formalDeviceUi.test.mjs
    printf 'SUMMARY result=pass mode=local precedence=deterministic gateway_dns=paired unreachable=actionable rollback=verified A=not-connected\n'
}

run_device() {
    [ "${M57_DEVICE_SMOKE:-0}" = 1 ] || {
        echo "device mode requires M57_DEVICE_SMOKE=1" >&2
        exit 2
    }
    command -v jq >/dev/null
    temp_dir="$(mktemp -d /tmp/quickstart-m57.XXXXXX)"
    cleanup() {
        rm -f "$temp_dir/targets.json"
        rmdir "$temp_dir" 2>/dev/null || true
    }
    trap cleanup EXIT HUP INT TERM

    a_before="$(fingerprint "$A_HOST")"
    b_before="$(fingerprint "$B_HOST")"
    remote "$A_HOST" "curl -fsS --max-time 10 -H 'X-Forwarded-Sid: ${SID}' 'http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2/gateway-targets/'" >"$temp_dir/targets.json"
    jq -e '
      (.result.targets | any(.kind == "floating" and .gateway == "192.168.30.3")) and
      (all(.result.targets[]; ((.gateway // "") == "") or ((.dns | length) == 1 and .dns[0] == .gateway)))
    ' "$temp_dir/targets.json" >/dev/null

    remote "$C_HOST" "ip -4 route show default | grep -Eq '^default via 192[.]168[.]30[.]244 '; ping -c 1 -W 2 192.168.30.244 >/dev/null"
    remote "$D_HOST" "ip -4 route show default | grep -Eq '^default via 192[.]168[.]30[.]1 '; ping -c 1 -W 2 192.168.30.1 >/dev/null"
    remote "$A_HOST" "ip -4 neigh show 192.168.30.244 | grep -vq FAILED"

    a_after="$(fingerprint "$A_HOST")"
    b_after="$(fingerprint "$B_HOST")"
    [ "$a_before" = "$a_after" ]
    [ "$b_before" = "$b_after" ]
    printf 'SUMMARY result=pass mode=device C_path=B D_path=A floating_target=available gateway_dns=paired A=read-only B=read-only config_drift=none\n'
}

case "$MODE" in
    local) run_local ;;
    device) run_device ;;
    *) echo "M57_SMOKE_MODE must be local or device" >&2; exit 2 ;;
esac
