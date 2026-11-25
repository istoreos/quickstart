#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

load_target_env
require_var ROLLBACK_RELEASE

if [ "$PROTECT_CRITICAL_GATEWAY" = 1 ] && [ "$ALLOW_CRITICAL_GATEWAY_DEPLOY" != 1 ]; then
    die "refusing rollback on protected critical gateway; set ALLOW_CRITICAL_GATEWAY_DEPLOY=1 for an explicitly approved rollback"
fi

remote_shell "
set -eu
backup='${REMOTE_BACKUP_DIR}/${ROLLBACK_RELEASE}'
# Keep the installed Unix controller paired with a compatible backend. Legacy
# backups need their matching controller restored through a paired migration.
if [ -d \"\${backup}\" ]; then
    rollback_binary=\"\${backup}/quickstart\"
else
    rollback_binary=\"\${backup}\"
fi
if ! grep -Fq '/var/run/quickstart/http.sock' /usr/lib/lua/luci/controller/istore_backend.lua || \\
   ! grep -Fq 'nixio.socket' /usr/lib/lua/luci/controller/istore_backend.lua || \\
   ! \"\${rollback_binary}\" serve --help 2>/dev/null | grep -Fq -- '--http-unix'; then
    echo 'rollback requires matching Unix socket backend and LuCI; restore legacy TCP pairs manually' >&2
    exit 1
fi
if [ -d \"\${backup}\" ]; then
    test -f \"\${backup}/quickstart\"
    test -d \"\${backup}/web\"
    test -f \"\${backup}/main.htm\"
    '${REMOTE_SERVICE}' stop || true
    cp -p \"\${backup}/quickstart\" '${REMOTE_BINARY}'
    chmod 0755 '${REMOTE_BINARY}'
    rm -rf '${REMOTE_WEB_DIR}'
    cp -a \"\${backup}/web\" '${REMOTE_WEB_DIR}'
    cp -p \"\${backup}/main.htm\" '${REMOTE_TEMPLATE}'
    if [ -f \"\${backup}/quickstart-state.keep\" ]; then
        cp -p \"\${backup}/quickstart-state.keep\" /lib/upgrade/keep.d/quickstart-state
    elif [ -f \"\${backup}/quickstart-state.keep.absent\" ]; then
        rm -f /lib/upgrade/keep.d/quickstart-state
    fi
    '${REMOTE_SERVICE}' start
else
    test -f \"\${backup}\"
    cp \"\${backup}\" '${REMOTE_BINARY}'
    chmod 0755 '${REMOTE_BINARY}'
    '${REMOTE_SERVICE}' restart
fi
sleep 1
'${REMOTE_SERVICE}' status
pidof quickstart >/dev/null
"

log "rollback ok: ${ROLLBACK_RELEASE}"
