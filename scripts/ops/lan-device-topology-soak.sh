#!/bin/sh
set -eu

duration_seconds="${1:-86400}"
sample_seconds="${2:-300}"
output_path="${3:-/tmp/lan-device-topology-soak.tsv}"

A_HOST="${A_HOST:-root@192.168.30.1}"
B_HOST="${B_HOST:-root@192.168.30.244}"
C_HOST="${C_HOST:-root@192.168.30.93}"
D_HOST="${D_HOST:-root@192.168.30.7}"

case "$duration_seconds:$sample_seconds" in
    *[!0-9:]*|:*|*:) echo "duration and sample interval must be positive integers" >&2; exit 2 ;;
esac
[ "$duration_seconds" -gt 0 ] && [ "$sample_seconds" -gt 0 ] || {
    echo "duration and sample interval must be positive" >&2
    exit 2
}

ssh_read() {
    host="$1"
    shift
    ssh -o BatchMode=yes -o ConnectTimeout=5 "$host" "$@" 2>/dev/null || printf 'unreachable'
}

owner_state() {
    value="$(ssh_read "$1" "ip -4 -o addr show dev br-lan | grep -q '192.168.30.3/' && printf present || printf absent")"
    printf '%s' "$value"
}

route_gateway() {
    ssh_read "$1" "ip route get 1.1.1.1 | awk '{ for (i=1; i<=NF; i++) if (\$i == \"via\") { print \$(i+1); exit } }'"
}

dns_server() {
    ssh_read "$1" "awk '\$1 == \"nameserver\" && \$2 ~ /^192\\.168\\.30\\./ { print \$2; exit }' /tmp/resolv.conf.d/resolv.conf.auto"
}

ping_state() {
    value="$(ssh_read "$1" "ping -c 1 -W 2 223.5.5.5 >/dev/null 2>&1 && printf ok || printf failed")"
    printf '%s' "$value"
}

mkdir -p "$(dirname "$output_path")"
printf 'utc\tepoch\ta_pid\ta_start_ticks\ta_rss_kb\ta_threads\tapi_http\tapi_seconds\ta_vip\tb_vip\tc_gateway\tc_dns\tc_ping\td_gateway\td_dns\td_ping\n' >"$output_path"

started_at="$(date +%s)"
deadline=$((started_at + duration_seconds))
while :; do
    now="$(date +%s)"
    a_process="$(ssh_read "$A_HOST" 'pid=$(pidof quickstart 2>/dev/null | awk '\''{print $1}'\''); [ -n "$pid" ] || { printf "missing missing 0 0"; exit; }; start=$(awk '\''{print $22}'\'' /proc/$pid/stat); rss=$(awk '\''$1 == "VmRSS:" {print $2}'\'' /proc/$pid/status); threads=$(awk '\''$1 == "Threads:" {print $2}'\'' /proc/$pid/status); printf "%s %s %s %s" "$pid" "$start" "${rss:-0}" "${threads:-0}"')"
    set -- $a_process
    a_pid="${1:-unreachable}"
    a_start="${2:-unreachable}"
    a_rss="${3:-0}"
    a_threads="${4:-0}"
    api="$(ssh_read "$A_HOST" "curl -sS --max-time 10 -o /dev/null -w '%{http_code} %{time_total}' -H 'X-Forwarded-Sid: t8-soak' http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2/devices/")"
    set -- $api
    api_http="${1:-000}"
    api_seconds="${2:-0}"
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
        "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$now" "$a_pid" "$a_start" "$a_rss" "$a_threads" \
        "$api_http" "$api_seconds" "$(owner_state "$A_HOST")" "$(owner_state "$B_HOST")" \
        "$(route_gateway "$C_HOST")" "$(dns_server "$C_HOST")" "$(ping_state "$C_HOST")" \
        "$(route_gateway "$D_HOST")" "$(dns_server "$D_HOST")" "$(ping_state "$D_HOST")" >>"$output_path"
    [ "$now" -ge "$deadline" ] && break
    sleep "$sample_seconds"
done
