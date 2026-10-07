#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "${SCRIPT_DIR}/../.." && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

load_target_env

: "${PLAYWRIGHT_NODE_PATH:=/config/playwright-runner/node_modules}"
: "${UI_AUDIT_SCRIPT:=${PROJECT_ROOT}/docs/evidence/m40/interaction-audit.cjs}"

ui_address="${UI_ADDRESS:-${SSH_TARGET#*@}}"
case "$ui_address" in
    ''|*[!A-Za-z0-9.:-]*) die "unsafe UI_ADDRESS: ${ui_address}" ;;
esac
: "${UI_URL:=http://${ui_address}/cgi-bin/luci/admin/quickstart/devicemanagement}"
: "${UI_OUTPUT:=/tmp/quickstart-ui-smoke-${ui_address}}"

command -v jq >/dev/null
command -v node >/dev/null
[ -f "$UI_AUDIT_SCRIPT" ] || die "UI audit script not found: ${UI_AUDIT_SCRIPT}"
[ -d "$PLAYWRIGHT_NODE_PATH/playwright" ] || die "Playwright dependency not found below: ${PLAYWRIGHT_NODE_PATH}"

reply="$(remote_shell "ubus -S call session create '{\"timeout\":300}'")"
sid="$(printf '%s' "$reply" | jq -r '.ubus_rpc_session // empty')"
case "$sid" in
    ''|*[!A-Fa-f0-9]*) die "remote session service returned an invalid identifier" ;;
esac

destroy_session() {
    remote_shell "ubus -S call session destroy '{\"ubus_rpc_session\":\"${sid}\"}' >/dev/null" >/dev/null 2>&1 || true
}
trap destroy_session EXIT HUP INT TERM

remote_shell "ubus -S call session set '{\"ubus_rpc_session\":\"${sid}\",\"values\":{\"username\":\"root\",\"token\":\"quickstart-ui-smoke\"}}' >/dev/null"
remote_shell "ubus -S call session grant '{\"ubus_rpc_session\":\"${sid}\",\"scope\":\"ubus\",\"objects\":[[\"*\",\"*\"]]}' >/dev/null"

NODE_PATH="$PLAYWRIGHT_NODE_PATH" \
M40_LUCI_COOKIE="$sid" \
M40_UI_URL="$UI_URL" \
M40_UI_OUTPUT="$UI_OUTPUT" \
    node "$UI_AUDIT_SCRIPT"

log "UI smoke ok: ${UI_URL} (report=${UI_OUTPUT}/report.json)"
