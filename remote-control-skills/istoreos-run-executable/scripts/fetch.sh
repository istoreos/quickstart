#!/bin/sh
set -eu

# fetch: download a URL to a local file (prefer KSpeeder DomainFold ksget if available).
#
# Usage:
#   fetch.sh -o <out_file> <URL>
#   fetch.sh <URL>                  # save to /tmp/istoreos-run-executable/<basename>
#
# Stdout: prints the saved file path only (for scripts to capture).

usage() {
  echo "usage: $0 [-o <out_file>] <URL>" >&2
}

out_file=""
while [ $# -gt 0 ]; do
  case "$1" in
    -o)
      shift
      [ $# -gt 0 ] || { usage; exit 2; }
      out_file="$1"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      break
      ;;
    -*)
      echo "unsupported option: $1" >&2
      usage
      exit 2
      ;;
    *)
      break
      ;;
  esac
done

url="${1:-}"
[ -n "$url" ] || { usage; exit 2; }

tmp_dir="${RUN_TMP_DIR:-/tmp/istoreos-run-executable}"
mkdir -p "$tmp_dir" 2>/dev/null || true

basename_from_url() {
  # strip query/fragment, then take last path segment
  u="$1"
  b="$(printf '%s' "$u" | sed 's/[?#].*$//' | awk -F/ '{print $NF}')"
  if [ -z "$b" ] || [ "$b" = "/" ]; then
    echo "download.run"
  else
    echo "$b"
  fi
}

skills_root() {
  if [ -n "${SKILLS_DIR:-}" ] && [ -d "$SKILLS_DIR" ]; then
    printf '%s\n' "$SKILLS_DIR"
    return 0
  fi

  here="$(CDPATH= cd "$(dirname "$0")" && pwd -P 2>/dev/null || pwd)"
  pack_root="$(dirname "$(dirname "$here")")"
  if [ -d "$pack_root/istoreos-run-executable" ]; then
    printf '%s\n' "$pack_root"
    return 0
  fi

  for dir in \
    "${CODEX_HOME:-/config/.codex}/skills" \
    "${CODEX_HOME:-$HOME/.codex}/skills" \
    "/config/.agents/skills" \
    "$HOME/.agents/skills" \
    "${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills" \
    "${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"
  do
    if [ -d "$dir" ]; then
      printf '%s\n' "$dir"
      return 0
    fi
  done

  printf '%s\n' "${CODEX_HOME:-$HOME/.codex}/skills"
}

if [ -z "$out_file" ]; then
  out_file="$tmp_dir/$(basename_from_url "$url")"
fi

part="${out_file}.part.$$"

SKILLS_DIR="$(skills_root)"

ksget=""
if [ -n "${SKILLS_DIR:-}" ] && [ -f "$SKILLS_DIR/istoreos-kspeeder-domainfold-fetch/scripts/ksget.sh" ]; then
  ksget="$SKILLS_DIR/istoreos-kspeeder-domainfold-fetch/scripts/ksget.sh"
fi

download_ok=0
if [ -n "$ksget" ]; then
  if sh "$ksget" -o "$part" "$url" >/dev/null 2>&1; then
    download_ok=1
  fi
fi

if [ "$download_ok" -ne 1 ]; then
  if command -v curl >/dev/null 2>&1; then
    curl -fL --retry 3 --connect-timeout 10 -o "$part" "$url" >/dev/null
    download_ok=1
  elif command -v wget >/dev/null 2>&1; then
    wget -O "$part" "$url" >/dev/null 2>&1
    download_ok=1
  elif command -v uclient-fetch >/dev/null 2>&1; then
    uclient-fetch -O "$part" "$url" >/dev/null 2>&1
    download_ok=1
  else
    echo "need: curl/wget/uclient-fetch not found" >&2
    exit 2
  fi
fi

if [ ! -s "$part" ]; then
  rm -f "$part" 2>/dev/null || true
  echo "download failed or empty: $url" >&2
  exit 1
fi

cat "$part" >"$out_file" 2>/dev/null || mv -f "$part" "$out_file"
rm -f "$part" 2>/dev/null || true

printf '%s\n' "$out_file"
