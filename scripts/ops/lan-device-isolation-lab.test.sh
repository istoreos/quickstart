#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
LAB_SCRIPT="$SCRIPT_DIR/lan-device-isolation-lab.sh"
TEST_ROOT="$(mktemp -d /tmp/quickstart-lab-test.XXXXXX)"
trap 'rm -f "$TEST_ROOT/output"; rmdir "$TEST_ROOT" 2>/dev/null || true' EXIT HUP INT TERM

LAN_LAB_ROOT="$TEST_ROOT/lab" "$LAB_SCRIPT" >"$TEST_ROOT/output"
grep -Fq 'fault_scenarios=12' "$TEST_ROOT/output"
grep -Fq 'A=not-connected' "$TEST_ROOT/output"
[ ! -e "$TEST_ROOT/lab" ]

set +e
LAN_LAB_MODE=remote-fixture LAN_LAB_MUTATION=1 \
    B_HOST=root@192.168.30.1 "$LAB_SCRIPT" > /dev/null 2>&1
critical_status=$?
set -e
[ "$critical_status" -eq 3 ]

grep -Fq 'LAN_LAB_MUTATION' "$LAB_SCRIPT"
grep -Fq 'refuse_critical_mutation "$B_HOST" "$C_HOST" "$D_HOST"' "$LAB_SCRIPT"
printf 'isolation lab safety and cleanup ok\n'
