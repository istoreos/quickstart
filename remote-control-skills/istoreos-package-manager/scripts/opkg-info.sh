#!/bin/sh
set -eu

kw="${1:-}"
limit="${2:-8}"

need() {
  echo "need: $*" >&2
  exit 2
}

have() { command -v "$1" >/dev/null 2>&1; }

if [ -z "$kw" ]; then
  echo "usage: $0 <pkg|regexp> [limit]" >&2
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

echo "## opkg info: $kw" >&2

out="$(opkg info "$kw" 2>/dev/null || true)"
if [ -n "$out" ]; then
  printf "%s\n" "$out"
  exit 0
fi

echo "warn: opkg info returned empty; try find top candidates then show info" >&2
pkgs="$(opkg find "$kw" 2>/dev/null | sed -n 's/^[[:space:]]*\\([^[:space:]]\\+\\)[[:space:]].*$/\\1/p' | head -n "$limit" || true)"
if [ -z "${pkgs:-}" ]; then
  pkgs="$(opkg list 2>/dev/null | grep -i -- "$kw" | sed -n 's/^[[:space:]]*\\([^[:space:]]\\+\\)[[:space:]]\\+-[[:space:]].*$/\\1/p' | head -n "$limit" || true)"
fi
if [ -z "${pkgs:-}" ]; then
  echo "no-candidates" >&2
  exit 1
fi

echo "$pkgs" | while IFS= read -r p; do
  [ -n "$p" ] || continue
  echo "" >&2
  echo "### pkg=$p" >&2
  opkg info "$p" 2>/dev/null || true
done
