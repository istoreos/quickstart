#!/bin/sh
set -eu

# Read-only by default. Set BANDIX_MUTATION_SMOKE=1 to create and immediately
# remove one high-limit rule for an unused locally-administered MAC address.
B_HOST="${B_HOST:-root@192.168.30.244}"
A_HOST="${A_HOST:-root@192.168.30.1}"
C_HOST="${C_HOST:-root@192.168.30.93}"
C_DEVICE_ID="${C_DEVICE_ID:-mac%3A46%3Af1%3Af3%3Ab9%3A9f%3Ab9}"
SID="${SID:-bandix-runtime-smoke}"
BANDIX_MUTATION_SMOKE="${BANDIX_MUTATION_SMOKE:-0}"
BANDIX_TEST_MAC="${BANDIX_TEST_MAC:-02:00:00:00:BE:EF}"
BANDIX_MAX_RSS_KB="${BANDIX_MAX_RSS_KB:-65536}"

checks=0
failures=0
rule_id=""
temp_dir="$(mktemp -d)"

remote() {
    brs_host="$1"
    shift
    ssh -o BatchMode=yes -o ConnectTimeout=8 "$brs_host" "$@"
}

cleanup() {
    if [ -n "$rule_id" ]; then
        remote "$B_HOST" "curl -fsS --max-time 10 -X DELETE -H 'Content-Type: application/json' --data-binary '{\"id\":\"${rule_id}\"}' 'http://127.0.0.1:8686/api/traffic/limits/schedule'" >/dev/null 2>&1 || true
    fi
    rm -r -- "$temp_dir"
}
trap cleanup EXIT HUP INT TERM

pass() {
    checks=$((checks + 1))
    printf 'PASS  %s\n' "$1"
}

fail() {
    checks=$((checks + 1))
    failures=$((failures + 1))
    printf 'FAIL  %s\n' "$1" >&2
}

expect_remote() {
    brs_label="$1"
    brs_host="$2"
    brs_command="$3"
    if remote "$brs_host" "$brs_command" >/dev/null 2>&1; then
        pass "$brs_label"
    else
        fail "$brs_label"
    fi
}

fetch_schedules() {
    remote "$B_HOST" "curl -fsS --max-time 10 'http://127.0.0.1:8686/api/traffic/limits/schedule'"
}

command -v ssh >/dev/null
command -v jq >/dev/null

case "$BANDIX_MUTATION_SMOKE:$B_HOST" in
    1:*192.168.30.1*)
        printf 'refusing Bandix mutation smoke on protected gateway %s\n' "$B_HOST" >&2
        exit 1
        ;;
esac

# A is the critical gateway: heartbeat only, with no package, service or config
# operations. This catches collateral network failures without adding load.
expect_remote 'A critical gateway reachable' "$A_HOST" 'true'
expect_remote 'A routing services healthy' "$A_HOST" '/etc/init.d/dnsmasq status >/dev/null && /etc/init.d/floatip status >/dev/null'

expect_remote 'B kernel major is at least 6' "$B_HOST" 'kernel=$(uname -r); major=${kernel%%.*}; test "$major" -ge 6'
expect_remote 'B kernel BTF is present' "$B_HOST" 'test -s /sys/kernel/btf/vmlinux'
expect_remote 'B bpffs is mounted' "$B_HOST" "grep -q ' /sys/fs/bpf bpf ' /proc/mounts"
expect_remote 'B scheduler BPF module is installed' "$B_HOST" "opkg list-installed kmod-sched-bpf | grep -q '^kmod-sched-bpf '"
expect_remote 'Bandix packages are installed' "$B_HOST" "opkg list-installed bandix | grep -q '^bandix ' && opkg list-installed luci-app-bandix | grep -q '^luci-app-bandix '"
expect_remote 'Bandix service is running and enabled' "$B_HOST" '/etc/init.d/bandix status >/dev/null && test -L /etc/rc.d/S99bandix'
expect_remote 'Bandix uses low-write traffic-only profile' "$B_HOST" "test \"\$(uci -q get bandix.traffic.enabled)\" = 1 && test \"\$(uci -q get bandix.traffic.traffic_enable_storage)\" = 0 && test \"\$(uci -q get bandix.connections.enabled)\" = 0 && test \"\$(uci -q get bandix.dns.enabled)\" = 0"
expect_remote 'Quickstart selects Bandix provider' "$B_HOST" "test \"\$(cat /etc/quickstart/rate-limit-provider)\" = bandix"
expect_remote 'Bandix traffic API returns devices' "$B_HOST" "curl -fsS --max-time 10 http://127.0.0.1:8686/api/traffic/devices | grep -q '\"status\":\"success\"'"
expect_remote 'Bandix process stays under RSS budget' "$B_HOST" "pid=\$(pgrep -o -f '^/usr/bin/bandix '); rss=\$(awk '\$1 == \"VmRSS:\" {print \$2}' /proc/\$pid/status); test \"\$rss\" -le ${BANDIX_MAX_RSS_KB}"
expect_remote 'C still routes through B and reaches Internet' "$C_HOST" "ip route get 1.1.1.1 | grep -q 'via 192.168.30.244 ' && ping -c 1 -W 2 223.5.5.5 >/dev/null"

fetch_schedules >"$temp_dir/schedules.json"
if jq -e '.status == "success" and (.data.limits | type == "array")' "$temp_dir/schedules.json" >/dev/null; then
    pass 'Bandix schedule API schema is compatible'
else
    fail 'Bandix schedule API schema is compatible'
fi

if remote "$B_HOST" "curl -fsS --max-time 10 -H 'X-Forwarded-Sid: ${SID}' 'http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2/device-policy/?deviceId=${C_DEVICE_ID}'" >"$temp_dir/policy.json" &&
    jq -e '.result.policy.rateLimit.provider == "bandix" and .result.policy.rateLimit.canApply == true and (.result.policy.rateLimit.state == "ready" or .result.policy.rateLimit.state == "verified")' "$temp_dir/policy.json" >/dev/null; then
    pass 'Quickstart observes Bandix enforcement for C'
else
    fail 'Quickstart observes Bandix enforcement for C'
fi

if [ "$BANDIX_MUTATION_SMOKE" = 1 ]; then
    if jq -e --arg mac "$(printf '%s' "$BANDIX_TEST_MAC" | tr 'A-F' 'a-f')" 'all(.data.limits[]; .mac != $mac)' "$temp_dir/schedules.json" >/dev/null; then
        pass 'test MAC has no pre-existing rule'
    else
        fail 'test MAC has no pre-existing rule'
    fi

    remote "$B_HOST" "curl -fsS --max-time 10 -X POST -H 'Content-Type: application/json' --data-binary '{\"mac\":\"${BANDIX_TEST_MAC}\",\"time_slot\":{\"start\":\"00:00\",\"end\":\"23:59\",\"days\":[1,2,3,4,5,6,7]},\"wan_tx_rate_limit\":125000000,\"wan_rx_rate_limit\":125000000}' 'http://127.0.0.1:8686/api/traffic/limits/schedule'" >"$temp_dir/create.json"
    rule_id="$(jq -r '.data.id // empty' "$temp_dir/create.json")"
    case "$rule_id" in
        ''|*[!A-Za-z0-9-]*)
            # If a future API omits the create response ID, recover it from the
            # unique test MAC so the EXIT trap can still remove the rule.
            fetch_schedules >"$temp_dir/create-recovery.json"
            rule_id="$(jq -r --arg mac "$(printf '%s' "$BANDIX_TEST_MAC" | tr 'A-F' 'a-f')" 'first(.data.limits[] | select(.mac == $mac) | .id) // empty' "$temp_dir/create-recovery.json")"
            ;;
    esac
    case "$rule_id" in
        ''|*[!A-Za-z0-9-]*)
            fail 'Bandix created a disposable rule'
            printf 'cannot identify disposable rule for safe cleanup\n' >&2
            exit 1
            ;;
        *) pass 'Bandix created a disposable rule' ;;
    esac

    fetch_schedules >"$temp_dir/created.json"
    if jq -e --arg id "$rule_id" 'any(.data.limits[]; .id == $id and .time_slot.end == "23:59" and .wan_tx_rate_limit == 125000000 and .wan_rx_rate_limit == 125000000)' "$temp_dir/created.json" >/dev/null; then
        pass 'Bandix returns the disposable rule accurately'
    else
        fail 'Bandix returns the disposable rule accurately'
    fi

    remote "$B_HOST" "curl -fsS --max-time 10 -X DELETE -H 'Content-Type: application/json' --data-binary '{\"id\":\"${rule_id}\"}' 'http://127.0.0.1:8686/api/traffic/limits/schedule'" >"$temp_dir/delete.json"
    rule_id=""
    fetch_schedules >"$temp_dir/deleted.json"
    if jq -e --arg mac "$(printf '%s' "$BANDIX_TEST_MAC" | tr 'A-F' 'a-f')" 'all(.data.limits[]; .mac != $mac)' "$temp_dir/deleted.json" >/dev/null; then
        pass 'Bandix disposable rule was removed'
    else
        fail 'Bandix disposable rule was removed'
    fi
fi

printf 'SUMMARY checks=%s failures=%s mutation=%s\n' "$checks" "$failures" "$BANDIX_MUTATION_SMOKE"
[ "$failures" -eq 0 ]
