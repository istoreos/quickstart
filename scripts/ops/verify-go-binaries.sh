#!/bin/sh
set -eu

: "${GO:=go}"
: "${EXPECTED_GO_VERSION:=1.26.11}"
: "${EXPECTED_GO_MODULE:=github.com/istoreos/quickstart/backend/cmd/backend}"

[ "$#" -gt 0 ] || {
    echo "usage: $0 BINARY..." >&2
    exit 2
}

for binary in "$@"; do
    [ -f "$binary" ] || {
        echo "error: Go binary not found: $binary" >&2
        exit 1
    }

    build_info="$($GO version -m "$binary")" || {
        echo "error: cannot read Go build info: $binary" >&2
        exit 1
    }
    go_version="$(printf '%s\n' "$build_info" | sed -n '1s/.*: go\([0-9][0-9.]*\)$/\1/p')"
    module_path="$(printf '%s\n' "$build_info" | sed -n 's/^[[:space:]]*path[[:space:]]*//p')"

    [ -n "$go_version" ] || {
        echo "error: missing Go version in build info: $binary" >&2
        exit 1
    }
    [ "$go_version" = "$EXPECTED_GO_VERSION" ] || {
        echo "error: $binary uses go${go_version}; release toolchain is pinned to go${EXPECTED_GO_VERSION}" >&2
        exit 1
    }
    [ "$module_path" = "$EXPECTED_GO_MODULE" ] || {
        echo "error: $binary module path is '$module_path'; expected '$EXPECTED_GO_MODULE'" >&2
        exit 1
    }

    printf '%s\n' "verified $binary: go${go_version} $module_path"
done
