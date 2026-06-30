#!/bin/sh
set -eu

MODE="${1:-}"
shift || true

help() {
  cat <<'EOF'
Usage:
  min-diag.sh [mode]

modes:
  base        - os + uname + df + mount (default)
  pkg         - is-opkg/opkg basics + relevant logread tail
  disk        - df -h/-i + mount
  docker      - docker version/info + uci dockerd/istoreenhance + dns + registry search

Notes:
  - Read-only commands only.
  - Output is intended to be pasted into support/AI.
EOF
}

say() {
  echo ""
  echo "## $1"
}

run() {
  echo "\$ $*"
  "$@" 2>&1 || true
}

case "$MODE" in
  ""|"base")
    say "Base"
    run cat /etc/os-release
    run uname -a
    run df -h
    run mount
    ;;
  "pkg")
    say "Package Tools"
    run is-opkg --help
    run opkg --help
    run opkg print-architecture
    say "Recent Logs (filtered)"
    run sh -c "logread 2>/dev/null | tail -n 200 | grep -iE 'opkg|ipkg|istore|install|upgrade|remove|error|fail' || true"
    ;;
  "disk")
    say "Disk"
    run df -h
    run df -i
    run mount
    ;;
  "docker")
    say "Docker"
    run docker version
    run sh -c "docker info 2>/dev/null | head -n 80 || true"
    say "UCI (dockerd/istoreenhance)"
    run sh -c "uci -q show dockerd 2>/dev/null | head -n 120 || true"
    run sh -c "uci -q show istoreenhance 2>/dev/null || true"
    say "DNS/Registry"
    host="${ISTORE_REGISTRY_HOST:-registry.linkease.net}"
    url="${ISTORE_REGISTRY_SEARCH_URL:-https://registry.linkease.net:5443/v1/search}"
    run sh -c "nslookup \"$host\" 2>/dev/null | head -n 60 || true"
    run sh -c "curl -fsSG --connect-timeout 2 --max-time 5 \"$url\" --data-urlencode 'q=busybox' --data 'n=1' 2>&1 | head -n 80 || true"
    ;;
  "-h"|"--help"|"help")
    help
    exit 0
    ;;
  *)
    echo "error: unknown mode: $MODE" >&2
    help >&2
    exit 2
    ;;
esac
