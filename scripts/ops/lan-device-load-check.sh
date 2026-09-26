#!/bin/sh
set -eu

BASE_URL="${BASE_URL:-http://127.0.0.1:3038}"
REQUESTS="${REQUESTS:-200}"
WORKERS="${WORKERS:-4}"
SID="${SID:-lan-device-load-check}"
PROTECT_CRITICAL_GATEWAY="${PROTECT_CRITICAL_GATEWAY:-1}"
CRITICAL_GATEWAY_ADDRESS="${CRITICAL_GATEWAY_ADDRESS:-192.168.30.1}"
ROUTER_ROLE="${ROUTER_ROLE:-}"
ROUTER_LOCAL_ADDRESS="${ROUTER_LOCAL_ADDRESS:-}"

case "$REQUESTS:$WORKERS" in
    *[!0-9:]*|0:*|*:0) echo "REQUESTS and WORKERS must be positive integers" >&2; exit 2 ;;
esac

if [ "$PROTECT_CRITICAL_GATEWAY" = 1 ] && { [ -z "$ROUTER_ROLE" ] || [ -z "$ROUTER_LOCAL_ADDRESS" ]; }; then
    context="$(curl -fsS --max-time 5 -H "X-Forwarded-Sid: ${SID}" \
        "${BASE_URL}/cgi-bin/luci/istore/lanctrl/v2/router-context/" 2>/dev/null || true)"
    [ -n "$ROUTER_ROLE" ] || ROUTER_ROLE="$(printf '%s' "$context" | sed -n 's/.*"topologyPosition":"\([^"]*\)".*/\1/p')"
    [ -n "$ROUTER_LOCAL_ADDRESS" ] || ROUTER_LOCAL_ADDRESS="$(printf '%s' "$context" | sed -n 's/.*"localAddress":"\([^"]*\)".*/\1/p')"
fi

if [ "$ROUTER_LOCAL_ADDRESS" = "$CRITICAL_GATEWAY_ADDRESS" ] || \
    { [ "$PROTECT_CRITICAL_GATEWAY" = 1 ] && [ "$ROUTER_ROLE" = lan_gateway_candidate ]; }; then
    echo "refusing load test on critical LAN gateway role=${ROUTER_ROLE:-unknown} address=${ROUTER_LOCAL_ADDRESS:-unknown}" >&2
    exit 3
fi

if [ "$PROTECT_CRITICAL_GATEWAY" = 1 ] && [ -z "$ROUTER_ROLE" ]; then
    echo "refusing load test because critical gateway role could not be determined" >&2
    exit 3
fi

quickstart_pid="$(pidof quickstart | awk '{print $1}')"
[ -n "$quickstart_pid" ] || { echo "quickstart is not running" >&2; exit 1; }

print_process_state() {
    printf 'pid=%s start_ticks=' "$quickstart_pid"
    awk '{print $22}' "/proc/${quickstart_pid}/stat"
    grep -E '^(VmRSS|VmHWM|Threads):' "/proc/${quickstart_pid}/status"
}

run_load() {
    label="$1"
    path="$2"
    per_worker=$(( (REQUESTS + WORKERS - 1) / WORKERS ))
    printf '%s ' "$label"
    {
        worker=0
        while [ "$worker" -lt "$WORKERS" ]; do
            (
                request=0
                while [ "$request" -lt "$per_worker" ]; do
                    curl -sS --max-time 10 -o /dev/null -w '%{http_code} %{time_total}\n' \
                        -H "X-Forwarded-Sid: ${SID}" "${BASE_URL}${path}" || true
                    request=$((request + 1))
                done
            ) &
            worker=$((worker + 1))
        done
        wait
    } | awk '
        BEGIN { min = 999; max = 0 }
        {
            count++
            if ($1 != 200) errors++
            sum += $2
            if ($2 < min) min = $2
            if ($2 > max) max = $2
        }
        END {
            if (count == 0) exit 2
            printf "requests=%d errors=%d min=%.4fs avg=%.4fs max=%.4fs\n", count, errors, min, sum/count, max
            if (errors > 0) exit 1
        }
    '
}

print_process_state
run_load devices '/cgi-bin/luci/istore/lanctrl/v2/devices/'
run_load router_context '/cgi-bin/luci/istore/lanctrl/v2/router-context/'
run_load targets '/cgi-bin/luci/istore/lanctrl/v2/gateway-targets/'
run_load policy '/cgi-bin/luci/istore/lanctrl/v2/device-network-policy/?deviceId=mac%3A46%3Af1%3Af3%3Ab9%3A9f%3Ab9'
sleep 10
print_process_state
