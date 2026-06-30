#!/bin/sh
set -eu

new="${1:-}"
confirm="${CONFIRM_DOCKER_DATA_ROOT_MIGRATE:-}"

need() {
  echo "need-confirmation: $*" >&2
  exit 2
}

fail() {
  echo "failed: $*" >&2
  exit 1
}

have() { command -v "$1" >/dev/null 2>&1; }

docker_root() {
  if have docker; then
    root="$(docker info 2>/dev/null | sed -n 's/^ Docker Root Dir: //p' | head -n 1 || true)"
    [ -n "${root:-}" ] && { echo "$root"; return 0; }
  fi
  if have uci; then
    root="$(uci -q get dockerd.globals.data_root 2>/dev/null || true)"
    [ -n "${root:-}" ] && { echo "$root"; return 0; }
  fi
  echo ""
}

ts() { date +%Y%m%d-%H%M%S; }

if [ "$confirm" != "YES" ]; then
  need "this is a risky operation (move docker data_root). Recommended: run a system backup first (istoreos-backup-restore). Rerun with CONFIRM_DOCKER_DATA_ROOT_MIGRATE=YES and provide <NEW_DATA_ROOT>."
fi

if [ -z "$new" ]; then
  need "missing <NEW_DATA_ROOT>; example: /mnt/<disk>/docker"
fi

case "$new" in
  /*) : ;;
  *) need "NEW_DATA_ROOT must be an absolute path: $new" ;;
esac

if printf '%s' "$new" | grep -q '^/overlay' 2>/dev/null; then
  need "refuse to migrate Docker data_root onto /overlay: $new (choose a data disk path under /mnt/...)"
fi

cur="$(docker_root)"
if [ -z "$cur" ]; then
  need "cannot detect current Docker data_root; run: docker info | sed -n '1,120p' and uci -q show dockerd | head -n 120"
fi

if [ "$cur" = "$new" ]; then
  echo "ok: already on requested data_root: $new" >&2
  exit 0
fi

if [ ! -x /etc/init.d/dockerd ]; then
  need "dockerd init script not found: /etc/init.d/dockerd"
fi

if ! have uci; then
  need "uci not found; cannot update /etc/config/dockerd safely"
fi

echo "action: stop dockerd" >&2
/etc/init.d/dockerd stop >/dev/null 2>&1 || true

echo "action: backup /etc/config/dockerd" >&2
bk="/etc/config/dockerd.bak.$(ts)"
cp -a /etc/config/dockerd "$bk" 2>/dev/null || true
echo "backup_dockerd=$bk" >&2

echo "action: create new root dir: $new" >&2
mkdir -p "$new" 2>/dev/null || fail "cannot create $new"

echo "action: migrate data (best-effort): $cur -> $new" >&2
if [ -d "$cur" ]; then
  if have rsync; then
    rsync -a --delete "$cur"/ "$new"/
  else
    echo "warn: rsync not found; using cp -a (may be slow and may not preserve all attributes)" >&2
    cp -a "$cur"/. "$new"/ 2>/dev/null || true
  fi
else
  echo "warn: current root not found as a directory: $cur (skipping copy)" >&2
fi

echo "action: set dockerd.globals.data_root='$new'" >&2
uci -q set "dockerd.globals.data_root=$new" || need "failed to set dockerd.globals.data_root; inspect /etc/config/dockerd"
uci -q commit dockerd || true

echo "action: start dockerd" >&2
/etc/init.d/dockerd start >/dev/null 2>&1 || /etc/init.d/dockerd restart >/dev/null 2>&1 || true

echo "## verify" >&2
if have docker; then
  docker info 2>/dev/null | sed -n '1,120p' || true
fi
df -hP "$new" 2>/dev/null || true

echo "## rollback" >&2
echo "uci -q set dockerd.globals.data_root='$cur' && uci -q commit dockerd && /etc/init.d/dockerd restart" >&2
echo "cp -a '$bk' /etc/config/dockerd && /etc/init.d/dockerd restart" >&2

echo "ok: migrate attempted (cur=$cur new=$new)" >&2
