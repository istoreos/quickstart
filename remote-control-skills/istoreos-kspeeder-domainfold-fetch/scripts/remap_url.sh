#!/bin/sh
set -eu

raw="${1:-}"
if [ -z "$raw" ]; then
  echo "usage: $0 <ORIGIN_URL>" >&2
  exit 2
fi

admin_port="${KSPEEDER_ADMIN_PORT:-5003}"

if ! command -v curl >/dev/null 2>&1; then
  echo "need-confirmation: curl not found; cannot call domainfold remap API." >&2
  exit 2
fi

json_escape() { printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'; }
payload="{\"url\":\"$(json_escape "$raw")\"}"

resp="$(curl -fsS -H "Content-Type: application/json" -d "$payload" "http://127.0.0.1:${admin_port}/api/domainfold/remap" 2>/dev/null || true)"
if [ -z "$resp" ]; then
  echo "failed: cannot query remap API at http://127.0.0.1:${admin_port}/api/domainfold/remap" >&2
  echo "hint: ensure iStoreEnhance is running and admin port is reachable" >&2
  exit 1
fi

json_get() {
  key="$1"
  printf '%s' "$resp" | tr -d '\n' | sed -n "s/.*\"$key\":\"\\([^\"]*\\)\".*/\\1/p" | head -n 1
}

out="$(json_get output)"
admin_path="$(json_get admin_path)"
err="$(json_get error)"

if [ -n "$err" ] && [ -z "$out" ]; then
  echo "failed: remap error: $err" >&2
  exit 1
fi

if [ -z "$out" ]; then
  echo "failed: remap output missing. raw response: $resp" >&2
  exit 1
fi

echo "$out"
if [ -n "$admin_path" ]; then
  echo "admin_proxy_url=http://127.0.0.1:${admin_port}${admin_path}" >&2
fi

