#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

version="$(project_version)"
[ -n "$version" ] || die "cannot read project version"

release_name="quickstart-binary-${version}"
release_dir="${BUILD_DIR}/${release_name}"
release_tarball="${BUILD_DIR}/${release_name}.tar.gz"
release_sha256="${release_tarball}.sha256"

"${SCRIPT_DIR}/build-backend.sh"
GOARCH=arm64 "${SCRIPT_DIR}/build-backend.sh"
GOARCH=arm GOARM=7 "${SCRIPT_DIR}/build-backend.sh"

rm -rf "$release_dir"
mkdir -p "$release_dir"
cp "${BUILD_DIR}/quickstart.arm64" "${release_dir}/quickstart.aarch64"
cp "${BUILD_DIR}/quickstart.amd64" "${release_dir}/quickstart.x86_64"
cp "${BUILD_DIR}/quickstart.armv7" "${release_dir}/quickstart.arm"

tar -C "$BUILD_DIR" -zcf "$release_tarball" "$release_name"
if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$release_tarball" | tee "$release_sha256"
else
    sha256sum "$release_tarball" | tee "$release_sha256"
fi

log "release tarball: ${release_tarball}"
log "release sha256: ${release_sha256}"
