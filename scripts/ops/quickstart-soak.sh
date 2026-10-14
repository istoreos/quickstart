#!/bin/sh
set -eu

duration_seconds="${1:-86400}"
sample_seconds="${2:-300}"
output_path="${3:-/root/quickstart-backups/quickstart-soak.tsv}"

case "$duration_seconds:$sample_seconds" in
	*[!0-9:]*|:*|*:) echo "duration and sample interval must be positive integers" >&2; exit 2 ;;
esac
[ "$duration_seconds" -gt 0 ] && [ "$sample_seconds" -gt 0 ] || {
	echo "duration and sample interval must be positive" >&2
	exit 2
}

started_at="$(date +%s)"
deadline=$((started_at + duration_seconds))
mkdir -p "$(dirname "$output_path")"
printf 'utc\tepoch\tpid\trss_kb\tvm_kb\tthreads\tcpu_ticks\tfd_count\tconfig_bytes\taudit_bytes\n' >"$output_path"

while :; do
	now="$(date +%s)"
	pid="$(pidof quickstart 2>/dev/null | awk '{print $1}')"
	if [ -z "$pid" ] || [ ! -r "/proc/$pid/status" ]; then
		printf '%s\t%s\tmissing\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$now" >>"$output_path"
		exit 1
	fi
	rss_kb="$(awk '$1 == "VmRSS:" { print $2 }' "/proc/$pid/status")"
	vm_kb="$(awk '$1 == "VmSize:" { print $2 }' "/proc/$pid/status")"
	threads="$(awk '$1 == "Threads:" { print $2 }' "/proc/$pid/status")"
	cpu_ticks="$(awk '{ print $14 + $15 }' "/proc/$pid/stat")"
	fd_count="$(ls "/proc/$pid/fd" 2>/dev/null | wc -l)"
	config_bytes="$(du -cb /etc/quickstart/*.json 2>/dev/null | awk '/total$/ { print $1 }')"
	audit_bytes="$(wc -c </etc/quickstart/network-audit.json 2>/dev/null || printf '0')"
	printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
		"$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$now" "$pid" "${rss_kb:-0}" \
		"${vm_kb:-0}" "${threads:-0}" "${cpu_ticks:-0}" "$fd_count" \
		"${config_bytes:-0}" "${audit_bytes:-0}" >>"$output_path"
	[ "$now" -ge "$deadline" ] && break
	sleep "$sample_seconds"
done

