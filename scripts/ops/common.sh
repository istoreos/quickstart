#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="${PROJECT_ROOT:-$(CDPATH= cd -- "${SCRIPT_DIR}/../.." && pwd)}"
BACKEND_ROOT="${BACKEND_ROOT:-${PROJECT_ROOT}/backend}"
BUILD_DIR="${BUILD_DIR:-${PROJECT_ROOT}/bin}"
VERSION_FILE="${VERSION_FILE:-${BACKEND_ROOT}/api/version.go}"
GO="${GO:-go}"

IT_RUNNER_DIR="${IT_RUNNER_DIR:-${PROJECT_ROOT}/.it-runner}"
ENV_DIR="${ENV_DIR:-${IT_RUNNER_DIR}/envs}"
SERVER_META_DIR="${SERVER_META_DIR:-${IT_RUNNER_DIR}/meta/servers}"
DEPLOYMENT_META_DIR="${DEPLOYMENT_META_DIR:-${IT_RUNNER_DIR}/meta/deployments}"

log() {
    printf '%s\n' "$*"
}

die() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

require_var() {
    var_name="$1"
    eval "var_value=\${$var_name-}"
    [ -n "$var_value" ] || die "missing required variable: $var_name"
}

load_env_file() {
    env_file="$1"
    [ -f "$env_file" ] || return 0
    set -a
    # shellcheck source=/dev/null
    . "$env_file"
    set +a
}

load_base_env() {
    load_env_file "${ENV_DIR}/000-defaults.env"
    load_env_file "${ENV_DIR}/010-local.env"
    : "${DEPLOY_TARGET:=quickstart-dev}"
}

load_target_env() {
    load_base_env

    deployment_file="${DEPLOYMENT_META_DIR}/${DEPLOY_TARGET}.env"
    [ -f "$deployment_file" ] || die "deployment target not found: ${DEPLOY_TARGET}"
    load_env_file "$deployment_file"

    require_var SERVER_REF
    server_file="${SERVER_META_DIR}/${SERVER_REF}.env"
    [ -f "$server_file" ] || die "server metadata not found: ${SERVER_REF}"
    load_env_file "$server_file"

    # Let machine-local overrides win over committed metadata.
    load_env_file "${ENV_DIR}/010-local.env"

    : "${SSH_PORT:=22}"
    : "${SSH_EXTRA_OPTS:=}"
    : "${REMOTE_BINARY:=/usr/sbin/quickstart}"
    : "${REMOTE_TMP:=/tmp/quickstart.new}"
    : "${REMOTE_BACKUP_DIR:=/tmp/quickstart-backups}"
    : "${REMOTE_SERVICE:=/etc/init.d/quickstart}"
    : "${REMOTE_LOG_COMMAND:=logread -e quickstart}"
    : "${LOCAL_BINARY:=${BUILD_DIR}/quickstart.amd64}"

    require_var SSH_TARGET
    require_var SSH_PORT
    require_var REMOTE_BINARY
    require_var REMOTE_TMP
    require_var REMOTE_BACKUP_DIR
    require_var REMOTE_SERVICE
    require_var REMOTE_LOG_COMMAND
}

project_version() {
    sed -n 's/^const VERSION = "\(.*\)"/\1/p' "$VERSION_FILE"
}

build_ldflags() {
    version="$(project_version)"
    [ -n "$version" ] || die "cannot read version from ${VERSION_FILE}"
    git_sha="$(git -C "$PROJECT_ROOT" rev-parse HEAD)"
    build_date="$(date -u +.%Y%m%d.%H%M%S)"
    printf '%s' "-X main.BuildVersion=${git_sha} -X main.BuildDate=${build_date} -X main.Version=${version} -s -w -extldflags \"-static\""
}

ssh_base_args() {
    printf '%s\n' "-p" "$SSH_PORT"
    if [ -n "${SSH_EXTRA_OPTS:-}" ]; then
        # Intentionally split operator-provided SSH options.
        # shellcheck disable=SC2086
        for opt in $SSH_EXTRA_OPTS; do
            printf '%s\n' "$opt"
        done
    fi
}

remote_shell() {
    # shellcheck disable=SC2046
    ssh $(ssh_base_args) "$SSH_TARGET" "$@"
}

copy_to_remote() {
    local_path="$1"
    remote_path="$2"
    # shellcheck disable=SC2046
    scp -P "$SSH_PORT" ${SSH_EXTRA_OPTS:-} "$local_path" "${SSH_TARGET}:${remote_path}"
}

ensure_local_binary() {
    [ -f "$LOCAL_BINARY" ] || die "local binary not found: ${LOCAL_BINARY}"
}
