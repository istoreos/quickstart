#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

load_target_env

if remote_shell "
set -eu
test -x '${REMOTE_BINARY}'
'${REMOTE_SERVICE}' status
pidof quickstart >/dev/null
"; then
    log "verify ok: ${SSH_TARGET}"
    exit 0
fi

log "verify failed; collecting remote diagnostics..." >&2
remote_shell "'${REMOTE_SERVICE}' status || true" || true
remote_shell "set +e; ${REMOTE_LOG_COMMAND}" || true
exit 1
