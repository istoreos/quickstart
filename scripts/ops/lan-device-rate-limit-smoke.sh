#!/bin/sh
set -eu

# P0-P2 rate-limit smoke. The only POST request is the documented read-only
# planning endpoint; this script never calls Apply or changes router services.
A_HOST="${A_HOST:-root@192.168.30.1}"
B_HOST="${B_HOST:-root@192.168.30.244}"
C_DEVICE_ID="${C_DEVICE_ID:-mac%3A46%3Af1%3Af3%3Ab9%3A9f%3Ab9}"
D_DEVICE_ID="${D_DEVICE_ID:-mac%3Abc%3A24%3A11%3A72%3A6d%3A38}"
SID="${SID:-lan-device-rate-limit-smoke}"

checks=0
failures=0
temp_dir="$(mktemp -d)"
trap 'rm -r -- "$temp_dir"' EXIT HUP INT TERM

pass() {
    checks=$((checks + 1))
    printf 'PASS  %s\n' "$1"
}

fail() {
    checks=$((checks + 1))
    failures=$((failures + 1))
    printf 'FAIL  %s\n' "$1" >&2
}

remote() {
    rl_host="$1"
    shift
    ssh -o BatchMode=yes -o ConnectTimeout=8 "$rl_host" "$@"
}

fetch() {
    rl_host="$1"
    rl_path="$2"
    rl_output="$3"
    remote "$rl_host" "curl -fsS --max-time 10 -H 'X-Forwarded-Sid: ${SID}' 'http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/${rl_path}'" >"$rl_output"
}

plan() {
    rl_host="$1"
    rl_payload="$2"
    rl_output="$3"
    printf '%s' "$rl_payload" | remote "$rl_host" \
        "curl -fsS --max-time 10 -H 'Content-Type: application/json' -H 'X-Forwarded-Sid: ${SID}' --data-binary @- 'http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2/rate-limit-settings/plan/'" \
        >"$rl_output"
}

config_digest() {
    remote "$1" 'sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall /etc/config/eqos /etc/config/floatip'
}

expect_jq() {
    rl_label="$1"
    rl_file="$2"
    rl_filter="$3"
    if jq -e "$rl_filter" "$rl_file" >/dev/null 2>&1; then
        pass "$rl_label"
    else
        fail "$rl_label"
    fi
}

command -v ssh >/dev/null
command -v jq >/dev/null

a_config_before="$(config_digest "$A_HOST")"
b_config_before="$(config_digest "$B_HOST")"

fetch "$A_HOST" globalConfigs/ "$temp_dir/a-global.json"
fetch "$B_HOST" globalConfigs/ "$temp_dir/b-global.json"
fetch "$A_HOST" "v2/device-policy/?deviceId=${D_DEVICE_ID}" "$temp_dir/a-policy.json"
fetch "$B_HOST" "v2/device-policy/?deviceId=${C_DEVICE_ID}" "$temp_dir/b-policy.json"

for node in a b; do
    global="$temp_dir/${node}-global.json"
    policy="$temp_dir/${node}-policy.json"
    host="$A_HOST"
    [ "$node" = b ] && host="$B_HOST"
    payload="$(jq -c '{settings: {
        enabled: (.result.capabilities.speedLimit.state == "available"),
        uploadSpeed: .result.speedLimit.uploadSpeed,
        downloadSpeed: .result.speedLimit.downloadSpeed,
        provider: .result.speedLimit.provider
    }}' "$global")"
    plan "$host" "$payload" "$temp_dir/${node}-plan.json"

    expect_jq "${node} provider registry" "$global" \
        '.result.speedLimit.provider as $active |
         (.result.speedLimit.providers | type == "array" and length >= 2) and
         any(.result.speedLimit.providers[]; .id == $active and (.available | type == "boolean")) and
         any(.result.speedLimit.providers[]; .id == "eqos") and
         any(.result.speedLimit.providers[]; .id == "bandix")'
    expect_jq "${node} unavailable providers explain why" "$global" \
        'all(.result.speedLimit.providers[]; .available == true or ((.reason // "") | length > 0))'
    expect_jq "${node} no-op plan is applicable and read-only" "$temp_dir/${node}-plan.json" \
        '.result.canApply == true and .result.error == null and
         ((.result.changes // []) | length == 0) and
         .result.current == .result.desired and
         ((.result.version // "") | length > 0) and .result.version == .result.rollbackPoint'
    expect_jq "${node} device enforcement contract" "$policy" \
        '.result.policy.rateLimit as $limit |
         ($limit | type == "object") and
         ($limit.provider | type == "string" and length > 0) and
         ($limit.executionNode | type == "string" and length > 0) and
         ($limit.state | type == "string" and length > 0) and
         ($limit.configured | type == "boolean") and
         ($limit.loaded | type == "boolean") and
         ($limit.verified | type == "boolean") and
         ($limit.canApply | type == "boolean") and
         ($limit.addressState | type == "string" and length > 0) and
         ($limit.ipv6State | type == "string" and length > 0) and
         ($limit.offloadState | type == "string" and length > 0)'
done

a_config_after="$(config_digest "$A_HOST")"
b_config_after="$(config_digest "$B_HOST")"
if [ "$a_config_before" = "$a_config_after" ]; then
    pass 'A planning smoke caused no configuration drift'
else
    fail 'A planning smoke caused no configuration drift'
fi
if [ "$b_config_before" = "$b_config_after" ]; then
    pass 'B planning smoke caused no configuration drift'
else
    fail 'B planning smoke caused no configuration drift'
fi

printf 'SUMMARY checks=%s failures=%s mode=read-only-plan\n' "$checks" "$failures"
[ "$failures" -eq 0 ]
