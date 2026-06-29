#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

: "${GOOS:=linux}"
: "${GOARCH:=amd64}"
: "${GOARM:=}"

mkdir -p "$BUILD_DIR"

case "$GOARCH${GOARM:+v${GOARM}}" in
    amd64) output="${BUILD_DIR}/quickstart.amd64" ;;
    arm64) output="${BUILD_DIR}/quickstart.arm64" ;;
    armv7) output="${BUILD_DIR}/quickstart.armv7" ;;
    *) die "unsupported GOARCH/GOARM: GOARCH=${GOARCH} GOARM=${GOARM}" ;;
esac

ldflags="$(build_ldflags)"
log "building ${output}"
(
    cd "$BACKEND_ROOT"
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" GOARM="$GOARM" \
        "$GO" build -a -ldflags "$ldflags" -o "$output" ./cmd/backend
)
