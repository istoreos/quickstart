#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

load_target_env
ensure_local_binary

local_sha="$(sha256sum "$LOCAL_BINARY" | awk '{print $1}')"
local_version="$("$LOCAL_BINARY" version --more 2>/dev/null || true)"

log "uploading ${LOCAL_BINARY} to ${SSH_TARGET}:${REMOTE_TMP}"
log "expected sha256: ${local_sha}"
[ -z "$local_version" ] || log "expected version: ${local_version}"

copy_to_remote "$LOCAL_BINARY" "$REMOTE_TMP"

remote_shell "
set -eu
stamp=\$(date +%Y%m%d%H%M%S)
mkdir -p '${REMOTE_BACKUP_DIR}'

if [ -x '${REMOTE_SERVICE}' ]; then
    '${REMOTE_SERVICE}' stop || true
fi

if [ -f '${REMOTE_BINARY}' ]; then
    cp '${REMOTE_BINARY}' '${REMOTE_BACKUP_DIR}/quickstart.\${stamp}.bak'
fi

cp '${REMOTE_TMP}' '${REMOTE_BINARY}'
chmod 0755 '${REMOTE_BINARY}'
rm -f '${REMOTE_TMP}'

if [ -x '${REMOTE_SERVICE}' ]; then
    '${REMOTE_SERVICE}' enable || true
    '${REMOTE_SERVICE}' restart || '${REMOTE_SERVICE}' start
fi

sleep 2

installed_sha=\$(sha256sum '${REMOTE_BINARY}' | awk '{print \$1}')
echo \"installed_sha=\${installed_sha}\"
if [ \"\${installed_sha}\" != '${local_sha}' ]; then
    echo 'installed quickstart sha mismatch' >&2
    exit 1
fi

pidof quickstart >/dev/null
echo 'deploy ok'
"

"${SCRIPT_DIR}/verify.sh"
