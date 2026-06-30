#!/bin/sh
set -eu

file="${1:-}"
confirm="${CONFIRM_ISTOREOS_RESTORE:-}"

need() {
  echo "need-confirmation: $*" >&2
  exit 2
}

if [ "$confirm" != "YES" ]; then
  need "this will restore system overlay and schedule reboot; rerun with CONFIRM_ISTOREOS_RESTORE=YES"
fi

if [ -z "$file" ]; then
  echo "usage: $0 <BACKUP_FILE>" >&2
  exit 2
fi

if [ ! -f "$file" ]; then
  need "backup file not found: $file"
fi

if [ ! -x /usr/libexec/istore/overlay-backup ]; then
  need "missing /usr/libexec/istore/overlay-backup"
fi

echo "action: restore from $file (will reboot after success)" >&2
/usr/libexec/istore/overlay-backup restore "$file"

