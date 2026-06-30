#!/bin/sh
set -eu

key="${1:-}"

have() { command -v "$1" >/dev/null 2>&1; }

need() {
  echo "need: $*" >&2
  exit 2
}

redact() {
  # Redact common secret-like keys in uci output:
  #   foo.bar.password='xxx'  -> foo.bar.password='***'
  #   foo.bar.token=xxx       -> foo.bar.token='***'
  sed -E \
    -e "s/((^|\\.)((password|passwd|token|secret|apikey|api_key|access_key|private_key|key))=)('[^']*'|[^[:space:]]+)/\\1'***'/gI" \
    -e "s/((^|\\.)((psk|auth|authorization))=)('[^']*'|[^[:space:]]+)/\\1'***'/gI"
}

section() {
  echo ""
  echo "## $*"
}

if [ -z "$key" ]; then
  echo "usage: $0 <name-or-keyword>" >&2
  exit 2
fi

echo "target=$key"

section "platform"
uname -a 2>/dev/null || true

section "paths"
ls -la "/etc/init.d/$key" "/etc/config/$key" 2>/dev/null || true
ls -la "/rom/etc/init.d/$key" "/rom/etc/config/$key" 2>/dev/null || true

section "uci"
if have uci; then
  uci -q show "$key" 2>/dev/null | redact | head -n 220 || true
else
  echo "uci=missing"
fi

section "init-script (head)"
if [ -f "/etc/init.d/$key" ]; then
  sed -n '1,220p' "/etc/init.d/$key" 2>/dev/null || true
else
  echo "missing=/etc/init.d/$key"
fi

section "init-script (rom head)"
if [ -f "/rom/etc/init.d/$key" ]; then
  sed -n '1,220p' "/rom/etc/init.d/$key" 2>/dev/null || true
else
  echo "missing=/rom/etc/init.d/$key"
fi

section "init-script diff (rom vs overlay)"
if [ -f "/rom/etc/init.d/$key" ] && [ -f "/etc/init.d/$key" ]; then
  if have diff; then
    diff -u "/rom/etc/init.d/$key" "/etc/init.d/$key" 2>/dev/null | head -n 160 || true
  else
    echo "diff=missing"
  fi
else
  echo "skip=need both /rom/etc/init.d/$key and /etc/init.d/$key"
fi

section "opkg"
if have opkg; then
  opkg status "app-meta-$key" 2>/dev/null | head -n 60 || true
  opkg status "$key" 2>/dev/null | head -n 60 || true
else
  echo "opkg=missing"
fi

section "luci (package files)"
if have opkg; then
  # best-effort: list luci files from packages whose name includes the key
  # (avoid expensive full filesystem scans on constrained devices)
  pkgs="$(opkg list-installed 2>/dev/null | awk '{print $1}' | grep -i "$key" | head -n 12 || true)"
  if [ -n "${pkgs:-}" ]; then
    echo "$pkgs" | while IFS= read -r p; do
      [ -n "$p" ] || continue
      echo "-- pkg=$p"
      opkg files "$p" 2>/dev/null | grep -E '^/usr/lib/lua/luci/' | head -n 120 || true
    done
  else
    echo "hint=no matching installed pkg names for keyword: $key"
  fi
else
  echo "skip=opkg missing"
fi

section "luci (filesystem match, small sample)"
if [ -d /usr/lib/lua/luci ]; then
  find /usr/lib/lua/luci -type f \( -name "*$key*" -o -path "*/$key/*" \) 2>/dev/null | head -n 60 || true
else
  echo "missing=/usr/lib/lua/luci"
fi

section "luci (rom exists)"
if [ -d /rom/usr/lib/lua/luci ]; then
  echo "rom_luci=present"
else
  echo "rom_luci=missing"
fi

section "next"
echo "1) If you need to change config or restart services, treat as risky: suggest full system backup first (istoreos-backup-restore)."
echo "2) If outputs include secrets, redact before sharing; only share minimal relevant sections."
