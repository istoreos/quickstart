#!/bin/sh
set -eu

say() { echo ""; echo "## $1"; }
run() { echo "\$ $*"; "$@" 2>&1 || true; }

say "istoreenhance Packages/Files"
run sh -c "opkg status app-meta-istoreenhance 2>/dev/null || true"
run sh -c "opkg status istoreenhance 2>/dev/null || true"
run ls -la /etc/init.d/istoreenhance
run ls -la /etc/config/istoreenhance
run ls -la /usr/sbin/iStoreEnhance

say "UCI"
run sh -c "uci -q show istoreenhance 2>/dev/null || true"
run sh -c "uci -q show dockerd 2>/dev/null | grep -F registry_mirrors || true"

say "Service/Process"
run sh -c "/etc/init.d/istoreenhance status 2>/dev/null || true"
run sh -c "ps w 2>/dev/null | grep -E '[i]StoreEnhance' || true"

say "DNS/Registry Search"
run sh -c "nslookup registry.linkease.net 2>/dev/null | head -n 60 || true"
run sh -c "curl -fsSG --connect-timeout 2 --max-time 5 'https://registry.linkease.net:5443/v1/search' --data-urlencode 'q=busybox' --data 'n=1' 2>&1 | head -n 80 || true"

