#!/bin/sh
set -eu

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

df_line() {
  p="$1"
  [ -n "$p" ] || return 0
  target="$p"
  [ -d "$target" ] || target="$(dirname "$target")"
  df -hP "$target" 2>/dev/null | tail -n 1 || true
}

skills_root() {
  dir="$(dirname "$0")"
  skill="$(dirname "$dir")"
  dirname "$skill"
}

suggest() {
  root="$(skills_root)"
  detect="$root/istoreos-storage-path/scripts/detect.sh"
  if [ -f "$detect" ]; then
    base="$(sh "$detect" 2>/dev/null || true)"
    [ -n "${base:-}" ] && echo "$base/docker"
  fi
}

root="$(docker_root)"
echo "## docker data_root"
if [ -n "$root" ]; then
  echo "current=$root"
  echo "df=$(df_line "$root")"
  if printf '%s' "$root" | grep -q '^/overlay' 2>/dev/null; then
    echo "warn=on_overlay (easy to fill system disk)" >&2
  fi
else
  echo "current=unknown (need docker running or uci dockerd.globals.data_root)" >&2
fi

new="$(suggest || true)"
if [ -n "${new:-}" ]; then
  echo ""
  echo "## suggested new root"
  echo "suggested=$new"
  root="$(skills_root)"
  echo "next=CONFIRM_DOCKER_DATA_ROOT_MIGRATE=YES sh \"$root/istoreos-docker-data-root-migrate/scripts/apply.sh\" '$new'"
else
  echo ""
  echo "## suggested new root"
  echo "suggested=unknown (cannot auto-pick data disk base path)" >&2
  echo "next=choose a data disk path like /mnt/<disk>/docker"
fi
