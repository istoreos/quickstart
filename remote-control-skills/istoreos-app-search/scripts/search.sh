#!/bin/sh
set -eu

q="${1:-}"
top="${2:-3}"
base="${ISTORE_AI_HELPER_BASE:-http://127.0.0.1:8197}"

need() {
  echo "need: $*" >&2
  exit 2
}

if [ -z "$q" ]; then
  echo "usage: $0 <keyword> [top]" >&2
  echo "env: ISTORE_AI_HELPER_BASE=http://127.0.0.1:8197" >&2
  exit 2
fi

case "$top" in
  ''|*[!0-9]*)
    need "top must be a positive integer: $top"
    ;;
esac

if ! command -v curl >/dev/null 2>&1; then
  need "curl not found; install curl or run search via another environment"
fi

curl -fsSG "$base/api/istore/app-search" \
  --data-urlencode "q=$q" \
  --data-urlencode "top=$top"

