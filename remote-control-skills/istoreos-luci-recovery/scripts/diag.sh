#!/bin/sh
set -eu

have() { command -v "$1" >/dev/null 2>&1; }

section() {
  echo ""
  echo "## $*"
}

redact() {
  sed -E \
    -e "s/((^|\\.)((password|passwd|token|secret|apikey|api_key|access_key|private_key|key))=)('[^']*'|[^[:space:]]+)/\\1'***'/gI" \
    -e "s/((^|\\.)((psk|auth|authorization))=)('[^']*'|[^[:space:]]+)/\\1'***'/gI"
}

echo "skill=istoreos-luci-recovery"

section "platform"
uname -a 2>/dev/null || true
if have ubus; then
  ubus call system board 2>/dev/null | head -n 80 || true
fi

section "disk"
df -hP / 2>/dev/null || true
if [ -d /overlay ]; then
  df -hP /overlay 2>/dev/null || true
fi

section "rom baseline"
test -d /rom && echo "rom=present" || echo "rom=missing"
ls -la /rom/etc/config 2>/dev/null | head -n 5 || true
ls -la /rom/etc/init.d 2>/dev/null | head -n 5 || true
test -d /rom/usr/lib/lua/luci && echo "rom_luci=present" || echo "rom_luci=missing"

section "services"
for s in rpcd uhttpd ubusd; do
  if [ -x "/etc/init.d/$s" ]; then
    echo "-- /etc/init.d/$s status"
    "/etc/init.d/$s" status 2>/dev/null || true
  else
    echo "-- missing=/etc/init.d/$s"
  fi
done

section "uci (uhttpd)"
if have uci; then
  uci -q show uhttpd 2>/dev/null | redact | head -n 220 || true
else
  echo "uci=missing"
fi

section "luci paths"
ls -la /usr/lib/lua/luci 2>/dev/null | head -n 40 || true
ls -la /www/cgi-bin/luci 2>/dev/null || true

section "opkg"
if have opkg; then
  echo "-- core status"
  for p in luci-base luci-mod-admin-full luci-lib-base rpcd uhttpd uhttpd-mod-ubus; do
    opkg status "$p" 2>/dev/null | head -n 30 || true
  done
  echo ""
  echo "-- installed luci packages (top)"
  opkg list-installed 2>/dev/null | grep -i '^luci' | head -n 120 || true
  echo ""
  echo "-- luci-related packages by keyword (top)"
  opkg list-installed 2>/dev/null | grep -iE 'luci|uhttpd|rpcd|ubus|cgi-io|lua' | head -n 160 || true
else
  echo "opkg=missing"
fi

section "logs (recent)"
if have logread; then
  logread 2>/dev/null | tail -n 260 | grep -iE 'uhttpd|rpcd|ubus|luci|lua|error|fail|traceback' || true
else
  echo "logread=missing"
fi

section "next"
echo "1) If you want a low-risk attempt: confirm soft recovery (clear /tmp luci cache + restart rpcd/uhttpd)."
echo "2) If still broken and disk is OK: confirm reinstall core packages (luci-base/uhttpd/rpcd...). Recommend full system backup first."

