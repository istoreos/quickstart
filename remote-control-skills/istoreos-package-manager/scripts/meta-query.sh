#!/bin/sh
set -eu

KW="${1:-}"
if [ -z "$KW" ]; then
  echo "usage: meta-query.sh <keyword>" >&2
  exit 2
fi

META_DIR="/usr/lib/opkg/meta"
if [ ! -d "$META_DIR" ]; then
  echo "meta dir not found: $META_DIR" >&2
  exit 1
fi

kw_lc="$(printf "%s" "$KW" | tr 'A-Z' 'a-z')"

echo "# matches for: $KW"
echo "# fields: name | title | entry | has_docker_deps | tags"
echo ""

for f in "$META_DIR"/*.json; do
  [ -f "$f" ] || continue
  content="$(cat "$f" 2>/dev/null || true)"
  [ -n "$content" ] || continue
  lc="$(printf "%s" "$content" | tr 'A-Z' 'a-z')"
  printf "%s" "$lc" | grep -q "$kw_lc" || continue

  name="$(printf "%s" "$content" | sed -n 's/.*"name"[ ]*:[ ]*"\([^"]*\)".*/\1/p' | head -n 1)"
  title="$(printf "%s" "$content" | sed -n 's/.*"title"[ ]*:[ ]*"\([^"]*\)".*/\1/p' | head -n 1)"
  entry="$(printf "%s" "$content" | sed -n 's/.*"entry"[ ]*:[ ]*"\([^"]*\)".*/\1/p' | head -n 1)"
  tags="$(printf "%s" "$content" | sed -n 's/.*"tags"[ ]*:[ ]*\\[\\([^]]*\\)\\].*/\\1/p' | head -n 1 | tr -d '\"' | tr ',' ' ')"
  has_docker="no"
  printf "%s" "$content" | grep -q '"docker-deps"' && has_docker="yes"

  printf "%s | %s | %s | %s | %s\n" "${name:-?}" "${title:-?}" "${entry:-}" "$has_docker" "${tags:-}"
done

