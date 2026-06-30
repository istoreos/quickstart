#!/bin/sh
set -eu

say() { echo ""; echo "## $1"; }
run() { echo "\$ $*"; "$@" 2>&1 || true; }

say "Binaries"
run sh -c "command -v docker 2>/dev/null || true"
run sh -c "command -v dockerd 2>/dev/null || true"

say "Service/Process"
run sh -c "/etc/init.d/dockerd status 2>/dev/null || true"
run sh -c "ps w 2>/dev/null | grep -E '[d]ockerd' || true"

say "Docker Version/Info"
run sh -c "docker version 2>&1 | head -n 120 || true"
run sh -c "docker info 2>&1 | head -n 80 || true"

say "Docker Space Check"
script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
skills_dir="$(CDPATH= cd -- "$script_dir/../.." && pwd)"
run sh -c "sh \"$skills_dir/istoreos-docker-basics/scripts/check_space.sh\" 2>&1 || true"

say "UCI dockerd"
run sh -c "uci -q show dockerd 2>/dev/null | head -n 120 || true"

say "Acceleration (istoreenhance/kspeeder) Quick Check"
run sh -c "opkg status app-meta-istoreenhance 2>/dev/null || true"
run sh -c "opkg status istoreenhance 2>/dev/null || true"
run sh -c "ls -la /etc/init.d/istoreenhance 2>/dev/null || true"
run sh -c "uci -q show istoreenhance 2>/dev/null || true"
run sh -c "uci -q show dockerd 2>/dev/null | grep -F registry_mirrors || true"
