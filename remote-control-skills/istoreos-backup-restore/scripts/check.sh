#!/bin/sh
set -eu

have() { command -v "$1" >/dev/null 2>&1; }

echo "## tools"
ls -la /usr/libexec/istore/backup 2>/dev/null || true
ls -la /usr/libexec/istore/overlay-backup 2>/dev/null || true
echo ""

echo "## sandbox"
if [ -d /ext_overlay ]; then
  echo "sandbox=present (/ext_overlay exists)"
else
  echo "sandbox=absent"
fi

echo ""
echo "## support"
if [ -x /usr/libexec/istore/overlay-backup ]; then
  /usr/libexec/istore/overlay-backup supports_overlay_backup 2>/dev/null || true
else
  echo "overlay-backup=missing"
fi

if [ -x /usr/libexec/istore/backup ]; then
  /usr/libexec/istore/backup get_support_backup_features 2>/dev/null || true
else
  echo "backup=missing"
fi

if have uci; then
  echo ""
  echo "## istore config"
uci -q show istore 2>/dev/null | head -n 80 || true
fi

