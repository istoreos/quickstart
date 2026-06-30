#!/bin/sh
set -eu

need() {
  echo "need-confirmation: $*" >&2
  exit 2
}

fail() {
  echo "failed: $*" >&2
  exit 1
}

have() { command -v "$1" >/dev/null 2>&1; }

confirm_apply() {
  [ "${CONFIRM_ISTOREENHANCE_APPLY:-}" = "YES" ] && return 0
  need "set CONFIRM_ISTOREENHANCE_APPLY=YES only after the user confirms package install/autoconf/UCI/service changes for istoreenhance."
}

installed() {
  [ -x /etc/init.d/istoreenhance ] || [ -x /usr/sbin/iStoreEnhance ] || have iStoreEnhance
}

skills_root() {
  dir="$(dirname "$0")"
  skill="$(dirname "$dir")"
  dirname "$skill"
}

autopick_base() {
  root="$(skills_root)"
  detect="$root/istoreos-storage-path/scripts/detect.sh"

  if [ -f "$detect" ]; then
    sh "$detect" || true
    return 0
  fi

  echo ""
}

pm_install() {
  pkg="$1"
  confirm_apply

  if have is-opkg; then
    is-opkg update >/dev/null 2>&1 || true
    is-opkg install "$pkg"
    return 0
  fi

  if have opkg; then
    opkg update >/dev/null 2>&1 || true
    opkg install "$pkg"
    return 0
  fi

  need "neither is-opkg nor opkg found; confirm how to install packages on this system."
}

cache_get() {
  if ! have uci; then
    echo ""
    return 0
  fi
  uci -q get 'istoreenhance.@istoreenhance[0].cache' 2>/dev/null || true
}

enabled_get() {
  if ! have uci; then
    echo ""
    return 0
  fi
  sec="$(uci -q show istoreenhance 2>/dev/null | sed -n "s/^istoreenhance\\.\\([^.=]*\\)=.*/\\1/p" | head -n 1 || true)"
  [ -n "$sec" ] || sec='@istoreenhance[0]'
  uci -q get "istoreenhance.${sec}.enabled" 2>/dev/null || true
}

enabled_set() {
  confirm_apply
  if ! have uci; then
    need "uci not found; cannot ensure istoreenhance is enabled in /etc/config/istoreenhance."
  fi
  sec="$(uci -q show istoreenhance 2>/dev/null | sed -n "s/^istoreenhance\\.\\([^.=]*\\)=.*/\\1/p" | head -n 1 || true)"
  if [ -z "$sec" ]; then
    sec="$(uci -q add istoreenhance istoreenhance 2>/dev/null || true)"
  fi
  if [ -z "$sec" ]; then
    need "cannot detect/create UCI section for istoreenhance; inspect /etc/config/istoreenhance and /etc/init.d/istoreenhance to confirm section naming."
  fi
  uci -q set "istoreenhance.${sec}.enabled=1" 2>/dev/null || return 1
  uci -q commit istoreenhance 2>/dev/null || return 1
  return 0
}

running() {
  if have pidof; then
    pidof iStoreEnhance >/dev/null 2>&1 && return 0
  fi
  if have pgrep; then
    pgrep -x iStoreEnhance >/dev/null 2>&1 && return 0
  fi
  return 1
}

if ! installed; then
  echo "action: install istoreenhance (kspeeder/iStoreEnhance) via iStore meta" >&2
  pm_install app-meta-istoreenhance 2>/dev/null || pm_install istoreenhance 2>/dev/null || true

  if ! installed; then
    need "istoreenhance still not detected after install attempt; please install manually (prefer: is-opkg install app-meta-istoreenhance) and reply '已安装'."
  fi
fi

cache="$(cache_get)"
base="${ISTOREENHANCE_BASE_PATH:-}"
if [ -z "$cache" ] && [ -n "$base" ] && have is-opkg; then
  confirm_apply
  echo "action: autoconf istoreenhance to base path: $base" >&2
  is-opkg AUTOCONF=istoreenhance "path=$base" enable=1 autoconf app-meta-istoreenhance >/dev/null 2>&1 || true
  cache="$(cache_get)"
fi

if [ -z "$cache" ] && [ -z "$base" ] && have is-opkg; then
  base="$(autopick_base)"
  if [ -n "$base" ]; then
    confirm_apply
    echo "action: autoconf istoreenhance to auto-picked base path: $base" >&2
    is-opkg AUTOCONF=istoreenhance "path=$base" enable=1 autoconf app-meta-istoreenhance >/dev/null 2>&1 || true
    cache="$(cache_get)"
  fi
fi

if [ -z "$cache" ]; then
  need "istoreenhance cache path not configured; pick a base path (via istoreos-storage-path), then run: is-opkg AUTOCONF=istoreenhance path=<base> enable=1 autoconf app-meta-istoreenhance (or set ISTOREENHANCE_BASE_PATH=<base> and rerun this script)"
fi

if [ ! -x /etc/init.d/istoreenhance ]; then
  need "service script not found: /etc/init.d/istoreenhance; confirm how istoreenhance is managed on this device."
fi

enabled="$(enabled_get)"
if [ "$enabled" != "1" ]; then
  echo "action: enable istoreenhance in UCI (/etc/config/istoreenhance)" >&2
  enabled_set || need "failed to set istoreenhance.@istoreenhance[0].enabled=1; please inspect /etc/init.d/istoreenhance to confirm enable flag and section naming."
fi

if ! running; then
  confirm_apply
  /etc/init.d/istoreenhance enable >/dev/null 2>&1 || true
  /etc/init.d/istoreenhance start >/dev/null 2>&1 || /etc/init.d/istoreenhance restart >/dev/null 2>&1 || true
fi

if ! running; then
  fail "istoreenhance not running after start; run: /etc/init.d/istoreenhance status && logread | tail -n 200 | grep -iE 'istoreenhance|iStoreEnhance|kspeeder|registry' "
fi

if have uci; then
  if ! uci -q show dockerd 2>/dev/null | grep -qF 'registry_mirrors' 2>/dev/null; then
    echo "warn: dockerd registry_mirrors not found in UCI; docker pulls may not use the local mirror." >&2
    echo "hint: check: uci -q show dockerd | head -n 120; then try: /etc/init.d/dockerd reload || /etc/init.d/dockerd restart" >&2
  fi
fi

echo "ok: istoreenhance installed/configured and running (cache=$cache)" >&2
