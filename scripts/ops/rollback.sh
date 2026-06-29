#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

load_target_env
require_var ROLLBACK_RELEASE

remote_shell "
set -eu
backup='${REMOTE_BACKUP_DIR}/${ROLLBACK_RELEASE}'
test -f \"\${backup}\"
cp \"\${backup}\" '${REMOTE_BINARY}'
chmod 0755 '${REMOTE_BINARY}'
'${REMOTE_SERVICE}' restart
sleep 2
'${REMOTE_SERVICE}' status
pidof quickstart >/dev/null
"

log "rollback ok: ${ROLLBACK_RELEASE}"
