#!/bin/sh
set -eu

dir="${1:-}"
need() {
  echo "need-confirmation: $*" >&2
  exit 2
}

if [ -z "$dir" ]; then
  echo "usage: $0 <DIR>" >&2
  exit 2
fi

case "$dir" in
  /|/tmp|/tmp/*|/overlay|/overlay/*|/ext_overlay|/ext_overlay/*)
    need "invalid backup dir: $dir (must be on external storage; refuse /, /tmp, /overlay, /ext_overlay)"
    ;;
esac

if [ ! -x /usr/libexec/istore/overlay-backup ]; then
  need "missing /usr/libexec/istore/overlay-backup (system full backup not supported on this device)"
fi

mp="$(findmnt -T "$dir" -o TARGET 2>/dev/null | sed -n 2p || true)"
if [ -n "${mp:-}" ] && [ "$mp" = "$dir" ]; then
  dir="$dir/istore_backup"
fi

mkdir -p "$dir" 2>/dev/null || need "cannot create backup dir: $dir"

echo "action: system backup to $dir" >&2
/usr/libexec/istore/overlay-backup backup "$dir"
