#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

load_target_env
ensure_local_candidate

if [ "$PROTECT_CRITICAL_GATEWAY" = 1 ] && [ "$ALLOW_CRITICAL_GATEWAY_DEPLOY" != 1 ]; then
    die "refusing deployment to protected critical gateway; set ALLOW_CRITICAL_GATEWAY_DEPLOY=1 for an explicitly approved rollout"
fi

for value_name in REMOTE_BINARY REMOTE_TMP REMOTE_WEB_ZIP REMOTE_INSTALLER REMOTE_BACKUP_DIR \
    REMOTE_SERVICE REMOTE_WEB_DIR REMOTE_TEMPLATE REMOTE_STAGE_ROOT REMOTE_HEALTH_URL; do
    validate_safe_deploy_value "$value_name"
done

local_sha="$(sha256sum "$LOCAL_BINARY" | awk '{print $1}')"
index_sha="$(unzip -p "$LOCAL_WEB_ZIP" luci-static/quickstart/index.js | sha256sum | awk '{print $1}')"
en_sha="$(unzip -p "$LOCAL_WEB_ZIP" luci-static/quickstart/i18n/en.json | sha256sum | awk '{print $1}')"
local_version="$("$LOCAL_BINARY" version --more 2>/dev/null || true)"
project_version="$(project_version)"
release_stamp="${DEPLOY_RELEASE:-${project_version}-$(git -C "$PROJECT_ROOT" rev-parse --short=12 HEAD)-$(date -u +%Y%m%d%H%M%S)}"
asset_version="${ASSET_VERSION:-${project_version}-candidate-$(printf '%s%s' "$index_sha" "$en_sha" | sha256sum | cut -c1-12)}"

case "$release_stamp:$asset_version" in
    *[!A-Za-z0-9._:-]*) die "release and asset versions must use only letters, digits, dot, underscore, colon, or dash" ;;
esac

log "uploading complete candidate to ${SSH_TARGET}"
log "expected sha256: ${local_sha}"
[ -z "$index_sha" ] || log "expected index sha256: ${index_sha}"
[ -z "$en_sha" ] || log "expected English catalog sha256: ${en_sha}"
[ -z "$local_version" ] || log "expected version: ${local_version}"
log "release: ${release_stamp}"
log "asset version: ${asset_version}"

copy_to_remote "$LOCAL_BINARY" "$REMOTE_TMP"
copy_to_remote "$LOCAL_WEB_ZIP" "$REMOTE_WEB_ZIP"
copy_to_remote "$LOCAL_REMOTE_INSTALLER" "$REMOTE_INSTALLER"

remote_shell "chmod 0755 '${REMOTE_INSTALLER}' && '${REMOTE_INSTALLER}' \
    '${release_stamp}' '${REMOTE_TMP}' '${REMOTE_WEB_ZIP}' '${REMOTE_BINARY}' \
    '${REMOTE_WEB_DIR}' '${REMOTE_TEMPLATE}' '${REMOTE_BACKUP_DIR}' '${REMOTE_SERVICE}' \
    '${local_sha}' '${index_sha}' '${en_sha}' '${asset_version}' '${REMOTE_HEALTH_URL}' \
    '${REMOTE_STAGE_ROOT}' '5' '/'"

remote_shell "rm -f '${REMOTE_INSTALLER}'"

EXPECTED_BINARY_SHA="$local_sha" EXPECTED_INDEX_SHA="$index_sha" EXPECTED_EN_SHA="$en_sha" \
    EXPECTED_ASSET_VERSION="$asset_version" "${SCRIPT_DIR}/verify.sh"
