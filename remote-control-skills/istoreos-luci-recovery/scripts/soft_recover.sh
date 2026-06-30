#!/bin/sh
set -eu

confirm="${CONFIRM_LUCI_SOFT_RECOVERY:-}"

need() {
  echo "need-confirmation: $*" >&2
  exit 2
}

have() { command -v "$1" >/dev/null 2>&1; }

if [ "$confirm" != "YES" ]; then
  need "this will clear LuCI caches under /tmp and restart rpcd/uhttpd (brief web UI interruption). Rerun with CONFIRM_LUCI_SOFT_RECOVERY=YES."
fi

echo "action: clear luci caches in /tmp" >&2
rm -rf \
  /tmp/luci-indexcache \
  /tmp/luci-indexcache.* \
  /tmp/luci-modulecache \
  /tmp/luci-modulecache.* \
  /tmp/luci-*cache* 2>/dev/null || true

echo "action: restart services (rpcd/uhttpd)" >&2
if [ -x /etc/init.d/rpcd ]; then
  /etc/init.d/rpcd restart >/dev/null 2>&1 || /etc/init.d/rpcd start >/dev/null 2>&1 || true
fi

if [ -x /etc/init.d/uhttpd ]; then
  /etc/init.d/uhttpd restart >/dev/null 2>&1 || /etc/init.d/uhttpd start >/dev/null 2>&1 || true
fi

echo "## status" >&2
if [ -x /etc/init.d/rpcd ]; then
  /etc/init.d/rpcd status 2>/dev/null || true
fi
if [ -x /etc/init.d/uhttpd ]; then
  /etc/init.d/uhttpd status 2>/dev/null || true
fi

echo "ok: soft recovery attempted" >&2
echo "next: retry web UI; if still broken, run diag.sh and consider reinstall_core.sh (after backup + confirmation)." >&2

