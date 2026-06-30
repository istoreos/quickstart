#!/bin/sh
set -eu

svc="${1:-}"
cfg="${2:-}"

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
  [ "${CONFIRM_SERVICE_APPLY:-}" = "YES" ] && return 0
  need "set CONFIRM_SERVICE_APPLY=YES only after the user confirms service enable/restart and UCI changes for svc=$svc cfg=$cfg."
}

if [ -z "$svc" ]; then
  echo "usage: $0 <service> [uci_config]" >&2
  exit 2
fi

if [ -z "$cfg" ]; then
  cfg="$svc"
fi

init="/etc/init.d/$svc"
conf="/etc/config/$cfg"

if [ ! -x "$init" ]; then
  need "init script not found: $init (confirm service name, or check /etc/init.d/)"
fi

if ! have uci; then
  echo "warn: uci not found; will only run init.d enable/start" >&2
fi

echo "action: init.d enable: $init enable" >&2
confirm_apply
"$init" enable >/dev/null 2>&1 || true

if have uci && [ -f "$conf" ]; then
  sec="$(uci -q show "$cfg" 2>/dev/null | sed -n "s/^$cfg\\.\\([^.=]*\\)=.*/\\1/p" | head -n 1 || true)"
  if [ -z "$sec" ]; then
    sec="@${cfg}[0]"
    echo "warn: cannot detect UCI section for $cfg; falling back to $sec" >&2
  fi

  enabled="$(uci -q get "${cfg}.${sec}.enabled" 2>/dev/null || true)"
  enable="$(uci -q get "${cfg}.${sec}.enable" 2>/dev/null || true)"
  disabled="$(uci -q get "${cfg}.${sec}.disabled" 2>/dev/null || true)"

  if [ -n "$enabled" ] || uci -q get "${cfg}.${sec}.enabled" >/dev/null 2>&1; then
    if [ "$enabled" != "1" ]; then
      echo "action: uci set ${cfg}.${sec}.enabled=1" >&2
      confirm_apply
      uci -q set "${cfg}.${sec}.enabled=1" || need "cannot set ${cfg}.${sec}.enabled=1; inspect $conf and $init"
      uci -q commit "$cfg" || true
    fi
  elif [ -n "$enable" ] || uci -q get "${cfg}.${sec}.enable" >/dev/null 2>&1; then
    if [ "$enable" != "1" ]; then
      echo "action: uci set ${cfg}.${sec}.enable=1" >&2
      confirm_apply
      uci -q set "${cfg}.${sec}.enable=1" || need "cannot set ${cfg}.${sec}.enable=1; inspect $conf and $init"
      uci -q commit "$cfg" || true
    fi
  elif [ -n "$disabled" ] || uci -q get "${cfg}.${sec}.disabled" >/dev/null 2>&1; then
    if [ "$disabled" != "0" ]; then
      echo "action: uci set ${cfg}.${sec}.disabled=0" >&2
      confirm_apply
      uci -q set "${cfg}.${sec}.disabled=0" || need "cannot set ${cfg}.${sec}.disabled=0; inspect $conf and $init"
      uci -q commit "$cfg" || true
    fi
  else
    echo "warn: no obvious enable flag found in $conf for ${cfg}.${sec} (enabled/enable/disabled)." >&2
    echo "hint: inspect $init for config_get_bool/config_get patterns to confirm how the service decides enabled state." >&2
  fi
fi

echo "action: init.d restart: $init restart" >&2
confirm_apply
"$init" restart >/dev/null 2>&1 || "$init" start >/dev/null 2>&1 || true

echo "## status ($svc)" >&2
"$init" status 2>&1 || true

echo "ok: ensured service attempted (svc=$svc cfg=$cfg)" >&2
