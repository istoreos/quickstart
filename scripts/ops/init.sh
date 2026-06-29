#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

load_target_env

remote_shell "
set -eu
mkdir -p '${REMOTE_BACKUP_DIR}'
test -d '${REMOTE_BACKUP_DIR}'
"

log "initialized ${SSH_TARGET}:${REMOTE_BACKUP_DIR}"
