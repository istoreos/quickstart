#!/bin/sh
set -eu

pattern="${1:-}"
limit="${2:-30}"

need() {
  echo "need: $*" >&2
  exit 2
}

have() { command -v "$1" >/dev/null 2>&1; }

if [ -z "$pattern" ]; then
  echo "usage: $0 <file|regexp> [limit]" >&2
  echo "example: $0 /usr/bin/curl" >&2
  exit 2
fi

case "$limit" in
  ''|*[!0-9]*)
    need "limit must be a positive integer: $limit"
    ;;
esac

if ! have opkg; then
  need "opkg not found"
fi

if [ "${CONFIRM_OPKG_UPDATE:-}" = "YES" ]; then
  echo "## opkg update (confirmed; writes package lists)" >&2
  opkg update >/dev/null 2>&1 || true
else
  echo "## opkg update skipped (set CONFIRM_OPKG_UPDATE=YES after user confirmation to refresh package lists)" >&2
fi

echo "## opkg search (file provider): $pattern" >&2
opkg search "$pattern" 2>/dev/null | head -n "$limit" || true
