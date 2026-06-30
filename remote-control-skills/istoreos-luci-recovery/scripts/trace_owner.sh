#!/bin/sh
set -eu

path="${1:-}"

have() { command -v "$1" >/dev/null 2>&1; }

need() {
  echo "need: $*" >&2
  exit 2
}

section() {
  echo ""
  echo "## $*"
}

sha() {
  f="$1"
  if have sha256sum; then
    sha256sum "$f" 2>/dev/null || true
  else
    echo "sha256sum=missing"
  fi
}

if [ -z "$path" ]; then
  echo "usage: $0 <FILE_PATH_UNDER_/>" >&2
  echo "example: $0 /usr/lib/lua/luci/dispatcher.lua" >&2
  exit 2
fi

case "$path" in
  /*) : ;;
  *) need "path must be absolute: $path" ;;
esac

section "paths"
if [ -f "$path" ]; then
  echo "overlay_file=present ($path)"
else
  echo "overlay_file=missing ($path)"
fi

rom="/rom$path"
if [ -f "$rom" ]; then
  echo "rom_file=present ($rom)"
else
  echo "rom_file=missing ($rom)"
fi

section "checksums"
if [ -f "$path" ]; then
  echo "-- overlay"
  sha "$path"
fi
if [ -f "$rom" ]; then
  echo "-- rom"
  sha "$rom"
fi

section "opkg owner"
if have opkg; then
  # `opkg search <path>` often returns lines like: pkg - /path
  # For busybox/older opkg, search may require exact path; we still try.
  opkg search "$path" 2>/dev/null | head -n 80 || true
else
  echo "opkg=missing"
fi

section "verify candidates"
if have opkg; then
  pkgs="$(opkg search "$path" 2>/dev/null | sed -n 's/^[[:space:]]*\\([^[:space:]]\\+\\)[[:space:]]\\+-[[:space:]].*$/\\1/p' | head -n 6 || true)"
  if [ -n "${pkgs:-}" ]; then
    echo "$pkgs" | while IFS= read -r p; do
      [ -n "$p" ] || continue
      echo "-- pkg=$p"
      opkg status "$p" 2>/dev/null | head -n 40 || true
      opkg files "$p" 2>/dev/null | grep -F "$path" || true
    done
  else
    echo "hint=no package owner found by opkg search for: $path"
    echo "hint=try reinstalling luci-base or the last installed luci-app/theme package"
  fi
fi

section "rom vs overlay diff (small)"
if [ -f "$path" ] && [ -f "$rom" ] && have diff; then
  diff -u "$rom" "$path" 2>/dev/null | head -n 120 || true
else
  echo "skip=need both files and diff tool"
fi

section "next"
echo "1) Prefer reinstall over editing: reinstall the owning package (or luci-base) and retry LuCI."
echo "2) If you must hotfix: back up first, keep a diff patch, and plan rollback to /rom or package reinstall."

