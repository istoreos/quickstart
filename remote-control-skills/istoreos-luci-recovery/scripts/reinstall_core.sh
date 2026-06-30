#!/bin/sh
set -eu

confirm="${CONFIRM_LUCI_REINSTALL:-}"

need() {
  echo "need-confirmation: $*" >&2
  exit 2
}

have() { command -v "$1" >/dev/null 2>&1; }

ts() { date +%Y%m%d-%H%M%S 2>/dev/null || echo now; }

run() {
  if have tee; then
    # shellcheck disable=SC2129
    echo "+ $*" >&2
    "$@" 2>&1 | tee -a "$LOG"
  else
    echo "+ $*" >&2
    "$@"
  fi
}

pm_update() {
  if have is-opkg; then
    run is-opkg update || true
    return 0
  fi
  if have opkg; then
    run opkg update || true
    return 0
  fi
  need "neither is-opkg nor opkg found; cannot reinstall LuCI packages."
}

pm_install_force() {
  pkg="$1"

  if have is-opkg; then
    run is-opkg install --force-reinstall "$pkg" && return 0
    run is-opkg install "$pkg" && return 0
  fi

  if have opkg; then
    run opkg install --force-reinstall "$pkg" && return 0
    run opkg install "$pkg" && return 0
  fi

  return 1
}

pkg_exists() {
  p="$1"
  if have opkg; then
    opkg info "$p" >/dev/null 2>&1 && return 0
  fi
  if have is-opkg; then
    is-opkg info "$p" >/dev/null 2>&1 && return 0
  fi
  return 1
}

if [ "$confirm" != "YES" ]; then
  need "this will update package lists and reinstall LuCI core packages (may take time and briefly interrupt web UI). Strongly recommended: do a full system backup first (istoreos-backup-restore). Rerun with CONFIRM_LUCI_REINSTALL=YES."
fi

LOG="/tmp/luci-reinstall.$(ts).log"
echo "log: $LOG" >&2

echo "action: package list update" >&2
pm_update

core="luci-base rpcd uhttpd uhttpd-mod-ubus"
optional="luci-mod-admin-full luci-lib-base luci-lib-ipkg luci-compat"

echo "action: reinstall core packages" >&2
for p in $core; do
  if pkg_exists "$p"; then
    echo "reinstall=$p" >&2
    pm_install_force "$p" || need "failed to install/reinstall: $p"
  else
    echo "skip=$p (not found in package lists)" >&2
  fi
done

echo "action: reinstall optional packages (if available)" >&2
for p in $optional; do
  if pkg_exists "$p"; then
    echo "reinstall=$p" >&2
    pm_install_force "$p" || true
  fi
done

echo "action: clear luci caches in /tmp" >&2
rm -rf \
  /tmp/luci-indexcache \
  /tmp/luci-indexcache.* \
  /tmp/luci-modulecache \
  /tmp/luci-modulecache.* \
  /tmp/luci-*cache* 2>/dev/null || true

echo "action: restart services (ubusd/rpcd/uhttpd)" >&2
for s in ubusd rpcd uhttpd; do
  if [ -x "/etc/init.d/$s" ]; then
    run "/etc/init.d/$s" restart || run "/etc/init.d/$s" start || true
  fi
done

echo "## status" >&2
for s in rpcd uhttpd; do
  if [ -x "/etc/init.d/$s" ]; then
    "/etc/init.d/$s" status 2>/dev/null || true
  fi
done

echo "ok: reinstall attempted" >&2
echo "next: retry web UI. If still broken, run diag.sh and identify the last installed/updated luci-app/theme package to rollback." >&2
