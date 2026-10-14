#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "${SCRIPT_DIR}/../.." && pwd)"
A_HOST="${A_HOST:-root@192.168.30.1}"
B_HOST="${B_HOST:-root@192.168.30.244}"
C_HOST="${C_HOST:-root@192.168.30.93}"
D_HOST="${D_HOST:-root@192.168.30.7}"
SID="${SID:-lan-device-business-smoke}"
C_DEVICE_ID="${C_DEVICE_ID:-mac%3A46%3Af1%3Af3%3Ab9%3A9f%3Ab9}"
D_DEVICE_ID="${D_DEVICE_ID:-mac%3Abc%3A24%3A11%3A72%3A6d%3A38}"

command -v ssh >/dev/null
command -v jq >/dev/null
command -v node >/dev/null

temp_dir="$(mktemp -d)"
trap 'rm -r -- "$temp_dir"' EXIT HUP INT TERM

remote() {
    bs_host="$1"
    shift
    ssh -o BatchMode=yes -o ConnectTimeout=8 "$bs_host" "$@"
}

fetch() {
    bs_host="$1"
    bs_resource="$2"
    bs_output="$3"
    bs_query="${4:-}"
    bs_url="http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2/${bs_resource}/"
    [ -z "$bs_query" ] || bs_url="${bs_url}?${bs_query}"
    remote "$bs_host" "curl -fsS --max-time 10 -H 'X-Forwarded-Sid: ${SID}' '${bs_url}'" >"$bs_output"
    jq -e '.result | type == "object"' "$bs_output" >/dev/null
}

collect_node() {
    bs_prefix="$1"
    bs_host="$2"
    fetch "$bs_host" router-context "$temp_dir/${bs_prefix}-router.json"
    fetch "$bs_host" devices "$temp_dir/${bs_prefix}-devices.json"
    fetch "$bs_host" gateway-targets "$temp_dir/${bs_prefix}-targets.json"
    fetch "$bs_host" floating-gateway "$temp_dir/${bs_prefix}-floating.json"
    fetch "$bs_host" lan-dhcp-settings "$temp_dir/${bs_prefix}-dhcp.json"
    fetch "$bs_host" network-rules "$temp_dir/${bs_prefix}-rules.json"
    fetch "$bs_host" device-groups "$temp_dir/${bs_prefix}-groups.json"
    fetch "$bs_host" traffic-insights "$temp_dir/${bs_prefix}-traffic.json" "deviceId=${C_DEVICE_ID}&range=today"
    fetch "$bs_host" device-network-policy "$temp_dir/${bs_prefix}-c-policy.json" "deviceId=${C_DEVICE_ID}"
}

# Critical gateway A is queried sequentially and read-only. This script contains
# no POST/Apply, service control, package operation, fault injection or load loop.
collect_node a "$A_HOST"
fetch "$A_HOST" device-network-policy "$temp_dir/a-d-policy.json" "deviceId=${D_DEVICE_ID}"
collect_node b "$B_HOST"

a_vip="$(remote "$A_HOST" "ip -4 -o addr show dev br-lan | grep -q '192.168.30.3/' && printf present || printf absent")"
b_vip="$(remote "$B_HOST" "ip -4 -o addr show dev br-lan | grep -q '192.168.30.3/' && printf present || printf absent")"
c_gateway="$(remote "$C_HOST" "ip route get 1.1.1.1 | sed -n 's/.* via \([^ ]*\).*/\1/p'")"
d_gateway="$(remote "$D_HOST" "ip route get 1.1.1.1 | sed -n 's/.* via \([^ ]*\).*/\1/p'")"
c_ping="$(remote "$C_HOST" 'ping -c 1 -W 2 223.5.5.5 >/dev/null 2>&1 && printf ok || printf failed')"
d_ping="$(remote "$D_HOST" 'ping -c 1 -W 2 223.5.5.5 >/dev/null 2>&1 && printf ok || printf failed')"

jq -n \
    --slurpfile ar "$temp_dir/a-router.json" --slurpfile ad "$temp_dir/a-devices.json" \
    --slurpfile at "$temp_dir/a-targets.json" --slurpfile af "$temp_dir/a-floating.json" \
    --slurpfile ah "$temp_dir/a-dhcp.json" --slurpfile an "$temp_dir/a-rules.json" \
    --slurpfile ag "$temp_dir/a-groups.json" --slurpfile ai "$temp_dir/a-traffic.json" \
    --slurpfile ac "$temp_dir/a-c-policy.json" --slurpfile ap "$temp_dir/a-d-policy.json" \
    --slurpfile br "$temp_dir/b-router.json" --slurpfile bd "$temp_dir/b-devices.json" \
    --slurpfile bt "$temp_dir/b-targets.json" --slurpfile bf "$temp_dir/b-floating.json" \
    --slurpfile bh "$temp_dir/b-dhcp.json" --slurpfile bn "$temp_dir/b-rules.json" \
    --slurpfile bg "$temp_dir/b-groups.json" --slurpfile bi "$temp_dir/b-traffic.json" \
    --slurpfile bc "$temp_dir/b-c-policy.json" \
    --arg aVip "$a_vip" --arg bVip "$b_vip" --arg cGateway "$c_gateway" \
    --arg dGateway "$d_gateway" --arg cPing "$c_ping" --arg dPing "$d_ping" \
    '{
      a: {routerContext: $ar[0].result, devices: $ad[0].result.devices, targets: $at[0].result.targets,
          floating: $af[0].result, dhcp: $ah[0].result, rules: $an[0].result, groups: $ag[0].result,
          traffic: $ai[0].result, cPolicy: $ac[0].result, dPolicy: $ap[0].result},
      b: {routerContext: $br[0].result, devices: $bd[0].result.devices, targets: $bt[0].result.targets,
          floating: $bf[0].result, dhcp: $bh[0].result, rules: $bn[0].result, groups: $bg[0].result,
          traffic: $bi[0].result, cPolicy: $bc[0].result},
      network: {aVip: $aVip, bVip: $bVip, cGateway: $cGateway, dGateway: $dGateway, cPing: $cPing, dPing: $dPing}
    }' >"$temp_dir/snapshot.json"

node "${PROJECT_ROOT}/scripts/analyze-lan-device-business-smoke.mjs" "$temp_dir/snapshot.json"
