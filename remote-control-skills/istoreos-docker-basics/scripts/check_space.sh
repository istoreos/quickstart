#!/bin/sh
set -eu

min_bytes="${DOCKER_MIN_FREE_BYTES:-2147483648}"
allow_low="${DOCKER_ALLOW_LOW_SPACE:-0}"

need() {
  echo "need-confirmation: $*" >&2
  exit 2
}

warn() {
  echo "warn: $*" >&2
}

have() { command -v "$1" >/dev/null 2>&1; }

docker_root() {
  if have docker; then
    root="$(docker info 2>/dev/null | sed -n 's/^ Docker Root Dir: //p' | head -n 1 || true)"
    if [ -n "${root:-}" ]; then
      echo "$root"
      return 0
    fi
  fi

  if have uci; then
    root="$(uci -q get dockerd.globals.data_root 2>/dev/null || true)"
    if [ -n "${root:-}" ]; then
      echo "$root"
      return 0
    fi
  fi

  echo ""
}

avail_bytes() {
  p="$1"
  [ -n "$p" ] || return 1

  if [ -d "$p" ]; then
    target="$p"
  else
    target="$(dirname "$p")"
  fi

  kb="$(df -kP "$target" 2>/dev/null | awk 'NR==2{print $4}' | head -n 1 || true)"
  [ -n "${kb:-}" ] || return 1

  echo $((kb * 1024))
}

root="$(docker_root)"
if [ -z "$root" ]; then
  need "cannot detect DockerRootDir/data_root (need docker running or UCI dockerd.globals.data_root); run: docker info | sed -n '1,120p' and uci -q show dockerd | head -n 120"
fi

free="$(avail_bytes "$root" || true)"
if [ -z "${free:-}" ]; then
  need "cannot compute free space for $root; run: df -kP '$root' and paste output"
fi

if printf '%s' "$root" | grep -q '^/overlay' 2>/dev/null || printf '%s' "$root" | grep -q '^/overlay/upper' 2>/dev/null; then
  warn "Docker data root is on system overlay: $root (easy to fill system disk and break iStoreOS)"
fi

if [ "$free" -lt "$min_bytes" ] 2>/dev/null; then
  msg="Docker free space is below threshold (free=$free bytes < min=$min_bytes bytes) at $root; installs/pulls may fail and can fill the system disk."
  if [ "$allow_low" = "1" ]; then
    warn "$msg (override: DOCKER_ALLOW_LOW_SPACE=1)"
    exit 0
  fi
  need "$msg (fix: migrate Docker data_root to a data disk via istoreos-docker-data-root-migrate, or free space; override: DOCKER_ALLOW_LOW_SPACE=1)"
fi

echo "ok: docker space looks sufficient (root=$root free_bytes=$free min_bytes=$min_bytes)" >&2
