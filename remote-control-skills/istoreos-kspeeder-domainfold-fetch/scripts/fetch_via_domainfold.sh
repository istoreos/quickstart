#!/bin/sh
set -eu

raw="${1:-}"
tool="${2:-curl}"
shift 2 2>/dev/null || true

if [ -z "$raw" ]; then
  echo "usage: $0 <ORIGIN_URL> [curl|wget] [extra args...]" >&2
  exit 2
fi

admin_port="${KSPEEDER_ADMIN_PORT:-5003}"

if ! command -v curl >/dev/null 2>&1; then
  echo "need-confirmation: curl not found; required to call remap API." >&2
  exit 2
fi

json_escape() { printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'; }
payload="{\"url\":\"$(json_escape "$raw")\"}"

resp="$(curl -fsS -H "Content-Type: application/json" -d "$payload" "http://127.0.0.1:${admin_port}/api/domainfold/remap" 2>/dev/null || true)"
if [ -z "$resp" ]; then
  echo "failed: cannot query remap API at http://127.0.0.1:${admin_port}/api/domainfold/remap" >&2
  echo "hint: run ensure_running.sh and confirm admin port" >&2
  exit 1
fi

admin_path="$(printf '%s' "$resp" | tr -d '\n' | sed -n 's/.*"admin_path":"\\([^"]*\\)".*/\\1/p' | head -n 1)"
err="$(printf '%s' "$resp" | tr -d '\n' | sed -n 's/.*"error":"\\([^"]*\\)".*/\\1/p' | head -n 1)"

if [ -n "$err" ] && [ -z "$admin_path" ]; then
  echo "failed: remap error: $err" >&2
  exit 1
fi
if [ -z "$admin_path" ]; then
  echo "failed: admin_path missing. raw response: $resp" >&2
  exit 1
fi

proxy_url="http://127.0.0.1:${admin_port}${admin_path}"
echo "info: domainfold admin proxy url: $proxy_url" >&2

case "$tool" in
  curl)
    exec curl -fL "$@" "$proxy_url"
    ;;
  wget)
    if command -v wget >/dev/null 2>&1; then
      exec wget "$@" "$proxy_url"
    fi
    if command -v uclient-fetch >/dev/null 2>&1; then
      exec uclient-fetch "$@" "$proxy_url"
    fi
    echo "need-confirmation: neither wget nor uclient-fetch found" >&2
    exit 2
    ;;
  *)
    echo "usage: $0 <ORIGIN_URL> [curl|wget] ..." >&2
    echo "unsupported tool: $tool" >&2
    exit 2
    ;;
esac

