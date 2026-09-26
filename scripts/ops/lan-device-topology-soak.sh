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

process_state() {
    ssh_read "$1" 'pid=$(pidof quickstart 2>/dev/null | awk '\''{print $1}'\''); [ -n "$pid" ] || { printf "missing missing 0 0 0"; exit; }; start=$(awk '\''{print $22}'\'' /proc/$pid/stat); rss=$(awk '\''$1 == "VmRSS:" {print $2}'\'' /proc/$pid/status); threads=$(awk '\''$1 == "Threads:" {print $2}'\'' /proc/$pid/status); fd=$(ls -1 /proc/$pid/fd 2>/dev/null | wc -l); printf "%s %s %s %s %s" "$pid" "$start" "${rss:-0}" "${threads:-0}" "${fd:-0}"'
}

api_state() {
    ssh_read "$1" "curl -sS --max-time 10 -o /dev/null -w '%{http_code} %{time_total}' -H 'X-Forwarded-Sid: m40-soak' http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2/devices/"
}

mkdir -p "$(dirname "$output_path")"
printf 'utc\tepoch\ta_pid\ta_start_ticks\ta_rss_kb\ta_threads\ta_fd\ta_api_http\ta_api_seconds\tb_pid\tb_start_ticks\tb_rss_kb\tb_threads\tb_fd\tb_api_http\tb_api_seconds\ta_vip\tb_vip\tc_gateway\tc_dns\tc_ping\td_gateway\td_dns\td_ping\n' >"$output_path"

started_at="$(date +%s)"
deadline=$((started_at + duration_seconds))
while :; do
    now="$(date +%s)"
    a_process="$(process_state "$A_HOST")"
    set -- $a_process
    a_pid="${1:-unreachable}"
    a_start="${2:-unreachable}"
    a_rss="${3:-0}"
    a_threads="${4:-0}"
    a_fd="${5:-0}"
    b_process="$(process_state "$B_HOST")"
    set -- $b_process
    b_pid="${1:-unreachable}"
    b_start="${2:-unreachable}"
    b_rss="${3:-0}"
    b_threads="${4:-0}"
    b_fd="${5:-0}"
    a_api="$(api_state "$A_HOST")"
    set -- $a_api
    a_api_http="${1:-000}"
    a_api_seconds="${2:-0}"
    b_api="$(api_state "$B_HOST")"
    set -- $b_api
    b_api_http="${1:-000}"
    b_api_seconds="${2:-0}"
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
        "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$now" "$a_pid" "$a_start" "$a_rss" "$a_threads" \
        "$a_fd" "$a_api_http" "$a_api_seconds" "$b_pid" "$b_start" "$b_rss" "$b_threads" "$b_fd" \
        "$b_api_http" "$b_api_seconds" "$(owner_state "$A_HOST")" "$(owner_state "$B_HOST")" \
        "$(route_gateway "$C_HOST")" "$(dns_server "$C_HOST")" "$(ping_state "$C_HOST")" \
        "$(route_gateway "$D_HOST")" "$(dns_server "$D_HOST")" "$(ping_state "$D_HOST")" >>"$output_path"
    [ "$now" -ge "$deadline" ] && break
    sleep "$sample_seconds"
done
