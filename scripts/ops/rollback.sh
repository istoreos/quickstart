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
