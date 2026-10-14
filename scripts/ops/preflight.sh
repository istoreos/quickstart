#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

load_target_env

log "checking ${SSH_TARGET}"
remote_shell "
set -eu
command -v sha256sum >/dev/null
command -v unzip >/dev/null
test -d /tmp
test -d '${REMOTE_WEB_DIR}'
test -f '${REMOTE_TEMPLATE}'
if [ -x '${REMOTE_SERVICE}' ]; then
    '${REMOTE_SERVICE}' status >/dev/null 2>&1 || true
fi
"

log "preflight ok: ${SSH_TARGET} (protected=${PROTECT_CRITICAL_GATEWAY})"
