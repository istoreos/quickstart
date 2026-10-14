#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

load_target_env

: "${EXPECTED_BINARY_SHA:=}"
: "${EXPECTED_INDEX_SHA:=}"
: "${EXPECTED_EN_SHA:=}"
: "${EXPECTED_ASSET_VERSION:=}"

if remote_shell "
set -eu
test -x '${REMOTE_BINARY}'
test -f '${REMOTE_WEB_DIR}/index.js'
test -f '${REMOTE_WEB_DIR}/i18n/en.json'
test -f '${REMOTE_TEMPLATE}'
'${REMOTE_SERVICE}' status
pidof quickstart >/dev/null
[ -z '${EXPECTED_BINARY_SHA}' ] || [ \"\$(sha256sum '${REMOTE_BINARY}' | cut -d ' ' -f1)\" = '${EXPECTED_BINARY_SHA}' ]
[ -z '${EXPECTED_INDEX_SHA}' ] || [ \"\$(sha256sum '${REMOTE_WEB_DIR}/index.js' | cut -d ' ' -f1)\" = '${EXPECTED_INDEX_SHA}' ]
[ -z '${EXPECTED_EN_SHA}' ] || [ \"\$(sha256sum '${REMOTE_WEB_DIR}/i18n/en.json' | cut -d ' ' -f1)\" = '${EXPECTED_EN_SHA}' ]
[ -z '${EXPECTED_ASSET_VERSION}' ] || grep -Fq 'local asset_version = \"${EXPECTED_ASSET_VERSION}\"' '${REMOTE_TEMPLATE}'
"; then
    log "verify ok: ${SSH_TARGET} (backend + web + asset version)"
    exit 0
fi

log "verify failed; collecting remote diagnostics..." >&2
remote_shell "'${REMOTE_SERVICE}' status || true" || true
remote_shell "set +e; ${REMOTE_LOG_COMMAND}" || true
exit 1
