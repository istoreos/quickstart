#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
: "${GO:=go}"
fixture="$(mktemp -d)"
trap 'rm -rf -- "$fixture"' EXIT

mkdir -p "$fixture/cmd/backend"
cat >"$fixture/go.mod" <<'EOF'
module github.com/istoreos/quickstart/backend

go 1.26.0
EOF
cat >"$fixture/cmd/backend/main.go" <<'EOF'
package main

func main() {}
EOF

(
    cd "$fixture"
    "$GO" build -trimpath -buildvcs=false -o "$fixture/quickstart" ./cmd/backend
)

EXPECTED_GO_VERSION=1.26.11 "$SCRIPT_DIR/verify-go-binaries.sh" "$fixture/quickstart" >/dev/null

if EXPECTED_GO_VERSION=99.0.0 "$SCRIPT_DIR/verify-go-binaries.sh" "$fixture/quickstart" >/dev/null 2>&1; then
    echo "verifier accepted an unpinned Go version" >&2
    exit 1
fi

if EXPECTED_GO_MODULE=example.invalid/backend "$SCRIPT_DIR/verify-go-binaries.sh" "$fixture/quickstart" >/dev/null 2>&1; then
    echo "verifier accepted an unexpected module path" >&2
    exit 1
fi

echo "Go binary verifier ok"
