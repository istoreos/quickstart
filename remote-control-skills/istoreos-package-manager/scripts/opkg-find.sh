#!/bin/sh
set -eu

kw="${1:-}"
limit="${2:-30}"

need() {
  echo "need: $*" >&2
  exit 2
}

have() { command -v "$1" >/dev/null 2>&1; }

supports() {
  cmd="$1"
  opkg --help 2>/dev/null | grep -qE "^[[:space:]]*$cmd[[:space:]]" || return 1
}

if [ -z "$kw" ]; then
  echo "usage: $0 <regexp> [limit]" >&2
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

echo "## opkg find (name/description match): $kw" >&2
if supports find; then
  opkg find "$kw" 2>/dev/null | head -n "$limit" || true
  exit 0
fi

echo "warn: opkg find not supported; fallback to list|grep (name-only)" >&2
opkg list 2>/dev/null | grep -i -- "$kw" | head -n "$limit" || true
