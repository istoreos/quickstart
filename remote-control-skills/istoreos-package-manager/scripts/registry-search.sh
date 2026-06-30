#!/bin/sh
set -eu

KW="${1:-}"
N="${2:-25}"
if [ -z "$KW" ]; then
  echo "usage: registry-search.sh <keyword> [n]" >&2
  exit 2
fi

URL="${ISTORE_REGISTRY_SEARCH_URL:-https://registry.linkease.net:5443/v1/search}"

curl -fsSG --connect-timeout 2 --max-time 8 \
  "$URL" \
  --data-urlencode "q=$KW" \
  --data "n=$N"
