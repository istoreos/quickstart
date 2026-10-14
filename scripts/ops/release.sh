#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
. "${SCRIPT_DIR}/common.sh"

version="$(project_version)"
[ -n "$version" ] || die "cannot read project version"

if [ "${ALLOW_DIRTY_RELEASE:-0}" != 1 ] && \
    { ! git -C "$PROJECT_ROOT" diff --quiet || ! git -C "$PROJECT_ROOT" diff --cached --quiet; }; then
    die "formal releases require a clean worktree; commit the candidate first or set ALLOW_DIRTY_RELEASE=1 for a local-only artifact"
fi

release_name="quickstart-binary-${version}"
release_dir="${BUILD_DIR}/${release_name}"
release_tarball="${BUILD_DIR}/${release_name}.tar.gz"
release_sha256="${release_tarball}.sha256"
source_date_epoch="${SOURCE_DATE_EPOCH:-$(git -C "$PROJECT_ROOT" show -s --format=%ct HEAD)}"
export SOURCE_DATE_EPOCH="$source_date_epoch"

"${SCRIPT_DIR}/build-backend.sh"
GOARCH=arm64 "${SCRIPT_DIR}/build-backend.sh"
GOARCH=arm GOARM=7 "${SCRIPT_DIR}/build-backend.sh"
npm --prefix "${PROJECT_ROOT}/web" run builds

rm -rf "$release_dir"
mkdir -p "$release_dir"
cp "${BUILD_DIR}/quickstart.arm64" "${release_dir}/quickstart.aarch64"
cp "${BUILD_DIR}/quickstart.amd64" "${release_dir}/quickstart.x86_64"
cp "${BUILD_DIR}/quickstart.armv7" "${release_dir}/quickstart.arm"
mkdir -p "${release_dir}/web"
cp -a "${PROJECT_ROOT}/web/dist/luci-static" "${release_dir}/web/luci-static"

binary_amd64_sha="$(sha256sum "${release_dir}/quickstart.x86_64" | awk '{print $1}')"
binary_arm64_sha="$(sha256sum "${release_dir}/quickstart.aarch64" | awk '{print $1}')"
binary_arm_sha="$(sha256sum "${release_dir}/quickstart.arm" | awk '{print $1}')"
index_sha="$(sha256sum "${release_dir}/web/luci-static/quickstart/index.js" | awk '{print $1}')"
en_sha="$(sha256sum "${release_dir}/web/luci-static/quickstart/i18n/en.json" | awk '{print $1}')"
asset_version="${version}-release-$(printf '%s%s' "$index_sha" "$en_sha" | sha256sum | cut -c1-12)"
printf '%s\n' \
    "version=${version}" \
    "git_sha=$(git -C "$PROJECT_ROOT" rev-parse HEAD)" \
    "asset_version=${asset_version}" \
    "quickstart.x86_64=${binary_amd64_sha}" \
    "quickstart.aarch64=${binary_arm64_sha}" \
    "quickstart.arm=${binary_arm_sha}" \
    "web.index.js=${index_sha}" \
    "web.i18n.en.json=${en_sha}" \
    >"${release_dir}/MANIFEST"

tar --sort=name --mtime="@${source_date_epoch}" --owner=0 --group=0 --numeric-owner \
    -C "$BUILD_DIR" -cf - "$release_name" | gzip -n >"$release_tarball"
if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$release_tarball" | tee "$release_sha256"
else
    sha256sum "$release_tarball" | tee "$release_sha256"
fi

log "release tarball: ${release_tarball}"
log "release sha256: ${release_sha256}"
