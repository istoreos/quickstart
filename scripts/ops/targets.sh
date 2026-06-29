#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

load_base_env

for deployment_file in "${DEPLOYMENT_META_DIR}"/*.env; do
    [ -f "$deployment_file" ] || continue
    target_name="$(basename "$deployment_file" .env)"
    SERVER_REF=
    DESCRIPTION=
    # shellcheck source=/dev/null
    . "$deployment_file"
    printf '%s\tserver=%s\t%s\n' "$target_name" "${SERVER_REF:-}" "${DESCRIPTION:-}"
done
