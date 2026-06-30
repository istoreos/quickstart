#!/bin/sh
set -eu

say() { echo "$*" >&2; }

ok() {
  base="$1"
  [ -n "$base" ] || return 1
  mkdir -p "$base" 2>/dev/null || return 1
  t="$base/.istore_write_test.$$"
  : >"$t" 2>/dev/null || return 1
  rm -f "$t" 2>/dev/null || true
  return 0
}

chosen=""
best_avail=0

cands="$(df -P -k 2>/dev/null | awk '
  NR==1 { next }
  $6 ~ "^/mnt/" || $6 ~ "^/media/" || $6 ~ "^/opt/" { print $6 "\t" $4 }
')"

tab="$(printf '\t')"
while IFS="$tab" read -r mp avail; do
  [ -n "${mp:-}" ] || continue
  [ -n "${avail:-}" ] || continue
  [ "$avail" -ge 1048576 ] 2>/dev/null || continue
  ok "$mp" || continue
  if [ "$avail" -gt "$best_avail" ] 2>/dev/null; then
    best_avail="$avail"
    chosen="$mp"
  fi
done <<EOF
$cands
EOF

if [ -n "$chosen" ]; then
  say "picked: base path=$chosen (largest writable mount under /mnt|/media|/opt, >= 1GiB free)"
  echo "$chosen"
  exit 0
fi

say "need-confirmation: no writable external/data mount under /mnt, /media, or /opt with >= 1GiB free."
say "need-confirmation: refusing to create or select a fallback path on the root overlay by default."
say "need-confirmation: choose an external/data mount path, or explicitly set an application-specific override after explaining overlay-full risk."
exit 2
