#!/bin/sh
set -eu

found=0

if command -v iStoreEnhance >/dev/null 2>&1; then
  echo "found: iStoreEnhance in PATH: $(command -v iStoreEnhance)" >&2
  found=1
fi

if [ -x /usr/sbin/iStoreEnhance ]; then
  echo "found: /usr/sbin/iStoreEnhance" >&2
  found=1
fi

if [ -x /etc/init.d/istoreenhance ]; then
  echo "found: /etc/init.d/istoreenhance" >&2
  found=1
fi

if [ "$found" -eq 1 ]; then
  exit 0
fi

echo "not installed: iStoreEnhance not found (PATH or /usr/sbin/iStoreEnhance), and /etc/init.d/istoreenhance not found." >&2
exit 1

