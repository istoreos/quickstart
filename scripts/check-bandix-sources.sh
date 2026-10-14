#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "${SCRIPT_DIR}/.." && pwd)"

check_source() {
    source_name="$1"
    source_path="$2"
    expected_commit="$3"
    expected_tag="$4"

    if [ ! -e "${PROJECT_ROOT}/${source_path}/.git" ]; then
        printf 'missing Bandix submodule: %s; run git submodule update --init --recursive\n' "$source_path" >&2
        return 1
    fi

    actual_commit="$(git -C "${PROJECT_ROOT}/${source_path}" rev-parse HEAD)"
    actual_tag="$(git -C "${PROJECT_ROOT}/${source_path}" describe --tags --exact-match 2>/dev/null || true)"
    if [ "$actual_commit" != "$expected_commit" ] || [ "$actual_tag" != "$expected_tag" ]; then
        printf '%s source mismatch: expected %s (%s), got %s (%s)\n' \
            "$source_name" "$expected_commit" "$expected_tag" "$actual_commit" "${actual_tag:-no-tag}" >&2
        return 1
    fi

    if ! git -C "${PROJECT_ROOT}/${source_path}" diff --quiet --ignore-submodules=none -- ||
        ! git -C "${PROJECT_ROOT}/${source_path}" diff --cached --quiet --ignore-submodules=none --; then
        printf '%s source has uncommitted changes: %s\n' "$source_name" "$source_path" >&2
        return 1
    fi

    printf 'PASS  %s %s %s\n' "$source_name" "$expected_tag" "$actual_commit"
}

check_source core third_party/bandix/core b28c8b52d973cbf6e607eec26f15c032c75b56cc v0.12.10
check_source openwrt third_party/bandix/openwrt 0cc5963fe009575fe20dfd1aa2dcf9b9f05f6712 v0.12.10
check_source luci third_party/bandix/luci 0bd9b61740c8798ca86317eb6e0784fb32dd51f5 v0.12.11
