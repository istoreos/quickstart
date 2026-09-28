#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
fixture="$(mktemp /tmp/quickstart-m64-soak.XXXXXX)"
trap 'rm -f "$fixture" "$fixture.bad"' EXIT HUP INT TERM
header='utc\tepoch\ta_pid\ta_start_ticks\ta_rss_kb\ta_threads\ta_fd\ta_api_http\ta_api_seconds\tb_pid\tb_start_ticks\tb_rss_kb\tb_threads\tb_fd\tb_api_http\tb_api_seconds\ta_vip\tb_vip\tc_gateway\tc_dns\tc_ping\td_gateway\td_dns\td_ping'
printf '%b\n' "$header" >"$fixture"
printf '%s\n' '2026-09-28T00:00:00Z\t100\t11\t20\t20000\t8\t12\t200\t0.02\t22\t30\t21000\t8\t13\t200\t0.03\tabsent\tpresent\t192.168.30.244\t192.168.30.244\tok\t192.168.30.1\t192.168.30.1\tok' | sed 's/\\t/\t/g' >>"$fixture"
printf '%s\n' '2026-09-29T00:00:00Z\t86500\t11\t20\t20500\t8\t13\t200\t0.02\t22\t30\t21400\t8\t14\t200\t0.03\tabsent\tpresent\t192.168.30.244\t192.168.30.244\tok\t192.168.30.1\t192.168.30.1\tok' | sed 's/\\t/\t/g' >>"$fixture"
"$SCRIPT_DIR/analyze-lan-device-topology-soak.sh" "$fixture" >/dev/null
sed 's/192\.168\.30\.244\t192\.168\.30\.244\tok/192.168.30.1\t192.168.30.244\tok/' "$fixture" >"$fixture.bad"
if "$SCRIPT_DIR/analyze-lan-device-topology-soak.sh" "$fixture.bad" >/dev/null 2>&1; then
    echo 'analyzer accepted a broken C route' >&2
    exit 1
fi
printf 'SUMMARY result=pass analyzer=duration-process-api-route-resource failure_fixture=rejected\n'
