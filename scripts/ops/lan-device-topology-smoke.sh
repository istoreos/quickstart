#!/bin/sh
set -eu

A_HOST="${A_HOST:-root@192.168.30.1}"
B_HOST="${B_HOST:-root@192.168.30.244}"
C_HOST="${C_HOST:-root@192.168.30.93}"
D_HOST="${D_HOST:-root@192.168.30.7}"
EXPECTED_VERSION="${EXPECTED_VERSION:-0.14.0}"
EXPECTED_PACKAGE_VERSION="${EXPECTED_PACKAGE_VERSION:-0.14.0-r12}"
EXPECTED_VIP_OWNER="${EXPECTED_VIP_OWNER:-B}"
REQUIRE_PACKAGE_COHERENCE="${REQUIRE_PACKAGE_COHERENCE:-1}"
CANDIDATE_MODE="${CANDIDATE_MODE:-0}"
PROJECT_ROOT="${PROJECT_ROOT:-$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)}"
LOCAL_CANDIDATE_BINARY="${LOCAL_CANDIDATE_BINARY:-${PROJECT_ROOT}/bin/quickstart.amd64}"
LOCAL_CANDIDATE_WEB_DIR="${LOCAL_CANDIDATE_WEB_DIR:-${PROJECT_ROOT}/web/dist/luci-static/quickstart}"
SID="${SID:-lan-device-topology-smoke}"

sha_file() {
    sha256sum "$1" | awk '{print $1}'
}

if [ "$CANDIDATE_MODE" = 1 ]; then
    for candidate_file in \
        "$LOCAL_CANDIDATE_BINARY" \
        "$LOCAL_CANDIDATE_WEB_DIR/index.js" \
        "$LOCAL_CANDIDATE_WEB_DIR/style.css" \
        "$LOCAL_CANDIDATE_WEB_DIR/vendor.js" \
        "$LOCAL_CANDIDATE_WEB_DIR/device-icons/manifest.json" \
        "$LOCAL_CANDIDATE_WEB_DIR/i18n/en.json"; do
        [ -f "$candidate_file" ] || {
            printf 'missing local candidate artifact: %s\n' "$candidate_file" >&2
            exit 1
        }
    done
    EXPECTED_BINARY_SHA="${EXPECTED_BINARY_SHA:-$(sha_file "$LOCAL_CANDIDATE_BINARY")}"
    EXPECTED_INDEX_SHA="${EXPECTED_INDEX_SHA:-$(sha_file "$LOCAL_CANDIDATE_WEB_DIR/index.js")}"
    EXPECTED_STYLE_SHA="${EXPECTED_STYLE_SHA:-$(sha_file "$LOCAL_CANDIDATE_WEB_DIR/style.css")}"
    EXPECTED_VENDOR_SHA="${EXPECTED_VENDOR_SHA:-$(sha_file "$LOCAL_CANDIDATE_WEB_DIR/vendor.js")}"
    EXPECTED_ICON_SHA="${EXPECTED_ICON_SHA:-$(sha_file "$LOCAL_CANDIDATE_WEB_DIR/device-icons/manifest.json")}"
    EXPECTED_EN_SHA="${EXPECTED_EN_SHA:-$(sha_file "$LOCAL_CANDIDATE_WEB_DIR/i18n/en.json")}"
    # The cache token is deployment metadata rather than a web artifact. If it
    # is not supplied, require both nodes to expose the same non-empty value.
    EXPECTED_ASSET_VERSION="${EXPECTED_ASSET_VERSION:-${ASSET_VERSION:-}}"
else
    EXPECTED_ASSET_VERSION="${EXPECTED_ASSET_VERSION:-0.14.0-r12}"
    EXPECTED_BINARY_SHA="${EXPECTED_BINARY_SHA:-1fea489dbba5077d9cdf72de2c3f0052d3143c555334ebe06bd2a2f119bf9653}"
    EXPECTED_INDEX_SHA="${EXPECTED_INDEX_SHA:-26ed35b4010558217ba7910f1b78ffc6b73f93623641fb42779f3b454c1af4b4}"
    EXPECTED_STYLE_SHA="${EXPECTED_STYLE_SHA:-96b1b3f5c2b9c48db1c937fc17f0f9f98e8fefd9569553c5c77355d88ef507f7}"
    EXPECTED_VENDOR_SHA="${EXPECTED_VENDOR_SHA:-ca04fc80ba23dd55bb5b5f5a9640ae21156b1b66f9759f60f48199db37cd99b8}"
    EXPECTED_ICON_SHA="${EXPECTED_ICON_SHA:-393f9be7ca9252a94646a8111814df2a9e07690cd5777392fab85053b36e7309}"
    EXPECTED_EN_SHA="${EXPECTED_EN_SHA:-2e0e34bc2f1e1eccb20e339ab4872f6a6d46b325127c4347df3376c4327b72e7}"
fi

failures=0
checks=0

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
    host="$1"
    shift
    ssh -o BatchMode=yes -o ConnectTimeout=8 "$host" "$@"
}

expect_remote() {
    er_label="$1"
    er_host="$2"
    er_command="$3"
    if remote "$er_host" "$er_command" >/dev/null 2>&1; then
        pass "$er_label"
    else
        fail "$er_label"
    fi
}

expect_value() {
    ev_label="$1"
    ev_expected="$2"
    ev_actual="$3"
    if [ "$ev_actual" = "$ev_expected" ]; then
        pass "$ev_label = $ev_actual"
    else
        fail "$ev_label expected=$ev_expected actual=${ev_actual:-<empty>}"
    fi
}

api_body() {
    ab_host="$1"
    ab_path="$2"
    remote "$ab_host" "curl -fsS --max-time 10 -H 'X-Forwarded-Sid: ${SID}' 'http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2/${ab_path}/'"
}

expect_api() {
    ea_label="$1"
    ea_host="$2"
    ea_path="$3"
    ea_filter="$4"
    ea_body="$(api_body "$ea_host" "$ea_path" 2>/dev/null || true)"
    if [ -n "$ea_body" ] && printf '%s' "$ea_body" | jq -e "$ea_filter" >/dev/null 2>&1; then
        pass "$ea_label"
    else
        fail "$ea_label"
    fi
}

runtime_checks() {
    rt_label="$1"
    rt_host="$2"
    expect_remote "$rt_label SSH reachable" "$rt_host" 'true'
    expect_remote "$rt_label Quickstart single process" "$rt_host" '[ "$(pidof quickstart | wc -w)" -eq 1 ] && /etc/init.d/quickstart status >/dev/null'
    expect_remote "$rt_label dnsmasq running" "$rt_host" '/etc/init.d/dnsmasq status >/dev/null'
    expect_remote "$rt_label floatip running" "$rt_host" '/etc/init.d/floatip status >/dev/null'

    expect_value "$rt_label backend version" "$EXPECTED_VERSION" "$(remote "$rt_host" '/usr/sbin/quickstart version' 2>/dev/null || true)"
    expect_value "$rt_label backend hash" "$EXPECTED_BINARY_SHA" "$(remote "$rt_host" "sha256sum /usr/sbin/quickstart | cut -d ' ' -f1" 2>/dev/null || true)"
    expect_value "$rt_label index hash" "$EXPECTED_INDEX_SHA" "$(remote "$rt_host" "sha256sum /www/luci-static/quickstart/index.js | cut -d ' ' -f1" 2>/dev/null || true)"
    expect_value "$rt_label style hash" "$EXPECTED_STYLE_SHA" "$(remote "$rt_host" "sha256sum /www/luci-static/quickstart/style.css | cut -d ' ' -f1" 2>/dev/null || true)"
    expect_value "$rt_label vendor hash" "$EXPECTED_VENDOR_SHA" "$(remote "$rt_host" "sha256sum /www/luci-static/quickstart/vendor.js | cut -d ' ' -f1" 2>/dev/null || true)"
    expect_value "$rt_label icon manifest hash" "$EXPECTED_ICON_SHA" "$(remote "$rt_host" "sha256sum /www/luci-static/quickstart/device-icons/manifest.json | cut -d ' ' -f1" 2>/dev/null || true)"
    expect_value "$rt_label English catalog hash" "$EXPECTED_EN_SHA" "$(remote "$rt_host" "sha256sum /www/luci-static/quickstart/i18n/en.json | cut -d ' ' -f1" 2>/dev/null || true)"
    actual_asset_version="$(remote "$rt_host" "sed -n 's/^local asset_version = \"\([^\"]*\)\"/\1/p' /usr/lib/lua/luci/view/quickstart/main.htm" 2>/dev/null || true)"
    if [ -n "$EXPECTED_ASSET_VERSION" ]; then
        expect_value "$rt_label asset version" "$EXPECTED_ASSET_VERSION" "$actual_asset_version"
    elif [ -n "$actual_asset_version" ]; then
        pass "$rt_label candidate asset version present"
    else
        fail "$rt_label candidate asset version present"
    fi

    quickstart_package="$(remote "$rt_host" "opkg list-installed quickstart | sed -n 's/^quickstart - //p'" 2>/dev/null || true)"
    luci_package="$(remote "$rt_host" "opkg list-installed luci-app-quickstart | sed -n 's/^luci-app-quickstart - //p'" 2>/dev/null || true)"
    if [ "$REQUIRE_PACKAGE_COHERENCE" = 1 ]; then
        expect_value "$rt_label quickstart package" "$EXPECTED_PACKAGE_VERSION" "$quickstart_package"
        expect_value "$rt_label LuCI package" "$EXPECTED_PACKAGE_VERSION" "$luci_package"
    else
        printf 'INFO  %s package quickstart=%s luci=%s (coherence gate disabled)\n' "$rt_label" "${quickstart_package:-missing}" "${luci_package:-missing}"
    fi

    expect_api "$rt_label devices API" "$rt_host" devices '.result.devices | type == "array"'
    expect_api "$rt_label gateway targets API" "$rt_host" gateway-targets '.result.health == "ready" and (.result.targets | type == "array")'
    expect_api "$rt_label floating gateway API" "$rt_host" floating-gateway '.result.status.capability == "available" and .result.status.serviceRunning == true'
    expect_api "$rt_label DHCP settings API" "$rt_host" lan-dhcp-settings '.result.settings | type == "object"'
    expect_api "$rt_label rules API" "$rt_host" network-rules '.result | type == "object"'
    expect_api "$rt_label groups API" "$rt_host" device-groups '.result | type == "object"'
    expect_api "$rt_label traffic API" "$rt_host" traffic-insights '.result | type == "object"'

    remote "$rt_host" 'pid=$(pidof quickstart); pid=${pid%% *}; printf "INFO  process pid=%s rss_kb=" "$pid"; awk '\''$1 == "VmRSS:" {printf "%s", $2}'\'' /proc/$pid/status; printf " threads="; awk '\''$1 == "Threads:" {printf "%s", $2}'\'' /proc/$pid/status; printf " fd="; ls -1 /proc/$pid/fd | wc -l' 2>/dev/null || true
}

command -v ssh >/dev/null
command -v jq >/dev/null

runtime_checks A "$A_HOST"
runtime_checks B "$B_HOST"

if [ "$CANDIDATE_MODE" = 1 ] && [ -z "$EXPECTED_ASSET_VERSION" ]; then
    a_asset_version="$(remote "$A_HOST" "sed -n 's/^local asset_version = \"\([^\"]*\)\"/\1/p' /usr/lib/lua/luci/view/quickstart/main.htm" 2>/dev/null || true)"
    b_asset_version="$(remote "$B_HOST" "sed -n 's/^local asset_version = \"\([^\"]*\)\"/\1/p' /usr/lib/lua/luci/view/quickstart/main.htm" 2>/dev/null || true)"
    expect_value 'candidate asset versions agree' "$a_asset_version" "$b_asset_version"
fi

expect_api 'A is LAN gateway with local DHCP authority' "$A_HOST" router-context '.result.topologyPosition == "lan_gateway_candidate" and .result.dhcpAuthority == "local" and .result.routeEditability.editable == true'
expect_api 'B is downstream and route editing is fail-closed' "$B_HOST" router-context '.result.topologyPosition == "downstream_router" and .result.dhcpAuthority != "local" and .result.routeEditability.editable == false'

a_vip="$(remote "$A_HOST" "ip -4 -o addr show dev br-lan | grep -q '192.168.30.3/' && printf present || printf absent" 2>/dev/null || true)"
b_vip="$(remote "$B_HOST" "ip -4 -o addr show dev br-lan | grep -q '192.168.30.3/' && printf present || printf absent" 2>/dev/null || true)"
if [ "$a_vip:$b_vip" = 'present:present' ] || [ "$a_vip:$b_vip" = 'absent:absent' ]; then
    fail "VIP must have exactly one owner (A=$a_vip B=$b_vip)"
else
    pass "VIP has exactly one owner (A=$a_vip B=$b_vip)"
fi
actual_owner=A
[ "$b_vip" = present ] && actual_owner=B
expect_value 'preferred VIP owner' "$EXPECTED_VIP_OWNER" "$actual_owner"

expect_remote 'C default route uses B' "$C_HOST" "ip route get 1.1.1.1 | grep -q 'via 192.168.30.244 '"
expect_remote 'D default route uses A' "$D_HOST" "ip route get 1.1.1.1 | grep -q 'via 192.168.30.1 '"
expect_remote 'C Internet reachable' "$C_HOST" 'ping -c 2 -W 2 223.5.5.5 >/dev/null'
expect_remote 'D Internet reachable' "$D_HOST" 'ping -c 2 -W 2 223.5.5.5 >/dev/null'
expect_remote 'C reaches floating gateway' "$C_HOST" 'ping -c 2 -W 2 192.168.30.3 >/dev/null'
expect_remote 'D reaches floating gateway' "$D_HOST" 'ping -c 2 -W 2 192.168.30.3 >/dev/null'

owner_host="$A_HOST"
[ "$actual_owner" = B ] && owner_host="$B_HOST"
owner_mac="$(remote "$owner_host" 'cat /sys/class/net/br-lan/address' 2>/dev/null || true)"
expect_remote 'C ARP resolves VIP to current owner' "$C_HOST" "ip neigh show 192.168.30.3 | grep -qi 'lladdr ${owner_mac}'"
expect_remote 'D ARP resolves VIP to current owner' "$D_HOST" "ip neigh show 192.168.30.3 | grep -qi 'lladdr ${owner_mac}'"

printf 'SUMMARY checks=%s failures=%s package_gate=%s candidate_mode=%s\n' "$checks" "$failures" "$REQUIRE_PACKAGE_COHERENCE" "$CANDIDATE_MODE"
[ "$failures" -eq 0 ]
