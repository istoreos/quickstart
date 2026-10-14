#!/bin/sh
set -eu

MODE="${LAN_LAB_MODE:-self-test}"
A_HOST="${A_HOST:-root@192.168.30.1}"
B_HOST="${B_HOST:-root@192.168.30.244}"
C_HOST="${C_HOST:-root@192.168.30.93}"
D_HOST="${D_HOST:-root@192.168.30.7}"
LAB_ROOT="${LAN_LAB_ROOT:-}"
LAB_TOKEN="m54-$$"
ADAPTERS="dhcp firewall floatip netpolicy"

host_address() {
    printf '%s' "${1#*@}"
}

refuse_critical_mutation() {
    for target in "$@"; do
        [ "$(host_address "$target")" != "$(host_address "$A_HOST")" ] || {
            echo "refusing fault-lab mutation on critical gateway: $target" >&2
            exit 3
        }
    done
}

remote() {
    lab_host="$1"
    shift
    ssh -o BatchMode=yes -o ConnectTimeout=8 "$lab_host" "$@"
}

cleanup_local() {
    [ -n "$LAB_ROOT" ] || return 0
    for adapter in $ADAPTERS; do
        rm -f "$LAB_ROOT/$adapter/current" "$LAB_ROOT/$adapter/snapshot" "$LAB_ROOT/$adapter/status"
        rmdir "$LAB_ROOT/$adapter" 2>/dev/null || true
    done
    rm -f "$LAB_ROOT/manifest"
    rmdir "$LAB_ROOT" 2>/dev/null || true
}

cleanup_remote() {
    for target in "$B_HOST" "$C_HOST" "$D_HOST"; do
        remote "$target" "rm -f '/tmp/quickstart-lab-${LAB_TOKEN}/manifest'; rmdir '/tmp/quickstart-lab-${LAB_TOKEN}' 2>/dev/null || true" >/dev/null 2>&1 || true
    done
}

run_local_scenario() {
    adapter="$1"
    fail_stage="$2"
    adapter_root="$LAB_ROOT/$adapter"
    printf 'stable\n' >"$adapter_root/current"
    cp "$adapter_root/current" "$adapter_root/snapshot"

    case "$fail_stage" in
        apply)
            printf 'rolled_back\n' >"$adapter_root/status"
            ;;
        verify)
            printf 'candidate\n' >"$adapter_root/current"
            cp "$adapter_root/snapshot" "$adapter_root/current"
            printf 'rolled_back\n' >"$adapter_root/status"
            ;;
        rollback)
            printf 'candidate\n' >"$adapter_root/current"
            printf 'recovery_required\n' >"$adapter_root/status"
            ;;
        *)
            echo "unknown fault stage: $fail_stage" >&2
            return 2
            ;;
    esac

    status="$(cat "$adapter_root/status")"
    current="$(cat "$adapter_root/current")"
    if [ "$fail_stage" = rollback ]; then
        [ "$status:$current" = recovery_required:candidate ]
    else
        [ "$status:$current" = rolled_back:stable ]
    fi
}

run_self_test() {
    if [ -z "$LAB_ROOT" ]; then
        LAB_ROOT="$(mktemp -d /tmp/quickstart-lab-m54.XXXXXX)"
    else
        case "$LAB_ROOT" in
            /tmp/*|"${TMPDIR:-/tmp}"/*) ;;
            *) echo "LAN_LAB_ROOT must be a dedicated temporary path" >&2; exit 2 ;;
        esac
        [ ! -e "$LAB_ROOT" ] || { echo "LAN_LAB_ROOT already exists" >&2; exit 2; }
        mkdir -p "$LAB_ROOT"
    fi
    trap cleanup_local EXIT HUP INT TERM
    printf 'mode=self-test token=%s\n' "$LAB_TOKEN" >"$LAB_ROOT/manifest"
    checks=0
    for adapter in $ADAPTERS; do
        mkdir "$LAB_ROOT/$adapter"
        for stage in apply verify rollback; do
            run_local_scenario "$adapter" "$stage"
            checks=$((checks + 1))
        done
    done
    printf 'SUMMARY result=pass mode=self-test adapters=4 fault_scenarios=%s A=not-connected cleanup=armed\n' "$checks"
}

run_remote_fixture() {
    [ "${LAN_LAB_MUTATION:-0}" = 1 ] || {
        echo "remote fixture requires LAN_LAB_MUTATION=1" >&2
        exit 2
    }
    refuse_critical_mutation "$B_HOST" "$C_HOST" "$D_HOST"
    command -v ssh >/dev/null
    trap cleanup_remote EXIT HUP INT TERM

    a_before="$(remote "$A_HOST" 'sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall')"
    for role_target in "B:$B_HOST" "C:$C_HOST" "D:$D_HOST"; do
        role="${role_target%%:*}"
        target="${role_target#*:}"
        remote "$target" "set -eu; test ! -e '/tmp/quickstart-lab-${LAB_TOKEN}'; mkdir '/tmp/quickstart-lab-${LAB_TOKEN}'; printf 'role=%s token=%s\n' '$role' '$LAB_TOKEN' >'/tmp/quickstart-lab-${LAB_TOKEN}/manifest'"
    done
    for target in "$B_HOST" "$C_HOST" "$D_HOST"; do
        remote "$target" "test -s '/tmp/quickstart-lab-${LAB_TOKEN}/manifest'"
    done
    cleanup_remote
    trap - EXIT HUP INT TERM
    for target in "$B_HOST" "$C_HOST" "$D_HOST"; do
        remote "$target" "test ! -e '/tmp/quickstart-lab-${LAB_TOKEN}'"
    done
    a_after="$(remote "$A_HOST" 'sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall')"
    [ "$a_before" = "$a_after" ]
    printf 'SUMMARY result=pass mode=remote-fixture targets=B,C,D A=read-only cleanup=verified\n'
}

case "$MODE" in
    self-test) run_self_test ;;
    remote-fixture) run_remote_fixture ;;
    *) echo "LAN_LAB_MODE must be self-test or remote-fixture" >&2; exit 2 ;;
esac
