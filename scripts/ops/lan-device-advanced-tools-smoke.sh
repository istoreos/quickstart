#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
MODE="${M63_SMOKE_MODE:-local}"
A_HOST="${A_HOST:-root@192.168.30.1}"
B_HOST="${B_HOST:-root@192.168.30.244}"
SID="${SID:-lan-device-m63-smoke}"
BASE="http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2"

remote() { host="$1"; shift; ssh -o BatchMode=yes -o ConnectTimeout=8 "$host" "$@"; }
api_get() { remote "$B_HOST" "curl -fsS --max-time 10 -H 'X-Forwarded-Sid: ${SID}' '${BASE}$1'"; }
api_post() { path="$1"; remote "$B_HOST" "curl -fsS --max-time 15 -H 'X-Forwarded-Sid: ${SID}' -H 'Content-Type: application/json' --data-binary @- '${BASE}${path}'"; }
fingerprint() { remote "$1" 'sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall 2>/dev/null; pidof dnsmasq; pidof quickstart'; }

run_local() {
    cd "$PROJECT_ROOT/backend"
    go test ./service -run 'Test(ManagementProbe|Webhook|DefaultWebhook|NetworkAudit|PolicyBundle|PolicyImport)' -count=1
    cd "$PROJECT_ROOT"
    node --test web/test/formalDeviceUi.test.mjs
    printf 'SUMMARY result=pass mode=local probe=inventory-only webhook=private-bounded-nonblocking import=plan-confirm-rollback\n'
}

run_device() {
    [ "${M63_DEVICE_SMOKE:-0}" = 1 ] || { echo 'device mode requires M63_DEVICE_SMOKE=1' >&2; exit 2; }
    command -v jq >/dev/null
    temp_dir="$(mktemp -d /tmp/quickstart-m63.XXXXXX)"
    mutated=0

    plan_bundle() {
        bundle="$1"; output="$2"
        jq -c '{bundle:.}' "$bundle" | api_post /policy-import/plan/ >"$output"
        jq -e '.result.plan.canApply == true and (.result.plan.checksum | length == 64)' "$output" >/dev/null
    }
    apply_bundle() {
        bundle="$1"; plan="$2"; output="$3"
        checksum="$(jq -r '.result.plan.checksum' "$plan")"
        version="$(jq -r '.result.plan.groupVersion' "$plan")"
        jq -c --arg checksum "$checksum" --arg version "$version" '{bundle:.,planChecksum:$checksum,expectedGroupVersion:$version,confirmed:true}' "$bundle" | api_post /policy-import/apply/ >"$output"
    }
    restore_original() {
        plan_bundle "$temp_dir/original-bundle.json" "$temp_dir/restore-plan.json"
        apply_bundle "$temp_dir/original-bundle.json" "$temp_dir/restore-plan.json" "$temp_dir/restore-apply.json"
        jq -e '.result.error == null' "$temp_dir/restore-apply.json" >/dev/null
        mutated=0
    }
    cleanup() {
        status="$?"
        trap - EXIT HUP INT TERM
        if [ "$mutated" = 1 ]; then
            restore_original >/dev/null 2>&1 || status=1
        fi
        rm -f "$temp_dir"/*
        rmdir "$temp_dir" 2>/dev/null || true
        exit "$status"
    }
    trap cleanup EXIT HUP INT TERM

    a_before="$(fingerprint "$A_HOST")"
    b_before="$(fingerprint "$B_HOST")"
    api_get /devices/ >"$temp_dir/devices.json"
    device_id="$(jq -r '.result.devices[] | select(any(.addresses.current[]?; .family == 4 and .address == "192.168.30.1")) | .deviceId' "$temp_dir/devices.json" | head -1)"
    [ -n "$device_id" ]

    jq -cn --arg id "$device_id" '{deviceId:$id,address:"192.168.30.1",scheme:"http",port:80}' | api_post /management-probe/ >"$temp_dir/probe-ok.json"
    jq -e '.result.reachable == true and (.result.statusCode >= 200 and .result.statusCode < 500) and .result.url == "http://192.168.30.1:80/"' "$temp_dir/probe-ok.json" >/dev/null
    for case_name in public_ip bad_port bad_scheme; do
        case "$case_name" in
            public_ip) payload="$(jq -cn --arg id "$device_id" '{deviceId:$id,address:"8.8.8.8",scheme:"http",port:80}')" ;;
            bad_port) payload="$(jq -cn --arg id "$device_id" '{deviceId:$id,address:"192.168.30.1",scheme:"http",port:22}')" ;;
            bad_scheme) payload="$(jq -cn --arg id "$device_id" '{deviceId:$id,address:"192.168.30.1",scheme:"file",port:80}')" ;;
        esac
        printf '%s' "$payload" | api_post /management-probe/ >"$temp_dir/probe-${case_name}.json"
        jq -e '.result.error.code == "validation_failed"' "$temp_dir/probe-${case_name}.json" >/dev/null
    done

    api_get "/advanced-network/?deviceId=${device_id}" >"$temp_dir/advanced-before.json"
    jq -e '.result.eventLimit > 0 and .result.eventLimit <= 512 and (.result.events | length) <= 20 and (.result.webhook.url == null or .result.webhook.url == "configured")' "$temp_dir/advanced-before.json" >/dev/null
    printf '%s' '{"enabled":true,"url":"http://user:pass@example.com/hook","events":["new_device"]}' | api_post /network-webhook/ >"$temp_dir/webhook-rejected.json"
    jq -e '.result.error.code == "validation_failed"' "$temp_dir/webhook-rejected.json" >/dev/null
    api_get "/advanced-network/?deviceId=${device_id}" >"$temp_dir/advanced-after.json"
    [ "$(jq -S -c '.result.webhook' "$temp_dir/advanced-before.json")" = "$(jq -S -c '.result.webhook' "$temp_dir/advanced-after.json")" ]

    api_get /policy-bundle/ >"$temp_dir/original-response.json"
    jq '.result.bundle' "$temp_dir/original-response.json" >"$temp_dir/original-bundle.json"
    jq -e '.schemaVersion == 1 and .scope == "device_groups_and_quotas" and (.groups|type)=="array" and (.devicePolicies|type)=="object" and (.quotas|type)=="object"' "$temp_dir/original-bundle.json" >/dev/null
    jq -e 'all(.groups[]; .id != "m63-smoke-temp")' "$temp_dir/original-bundle.json" >/dev/null
    original_normalized="$(jq -S -c 'del(.exportedAt)' "$temp_dir/original-bundle.json")"

    jq -c '{bundle:.}' "$temp_dir/original-bundle.json" | api_post /policy-import/apply/ >"$temp_dir/no-confirm.json"
    jq -e '.result.error.code == "confirmation_required"' "$temp_dir/no-confirm.json" >/dev/null
    jq '.schemaVersion=99' "$temp_dir/original-bundle.json" | jq -c '{bundle:.}' | api_post /policy-import/plan/ >"$temp_dir/bad-version.json"
    jq -e '.result.error.code == "incompatible_version"' "$temp_dir/bad-version.json" >/dev/null

    plan_bundle "$temp_dir/original-bundle.json" "$temp_dir/same-plan.json"
    apply_bundle "$temp_dir/original-bundle.json" "$temp_dir/same-plan.json" "$temp_dir/same-apply.json"
    jq -e '.result.error == null and (.result.changed // false) == false' "$temp_dir/same-apply.json" >/dev/null

    jq '.groups += [{id:"m63-smoke-temp",name:"M63 smoke temporary group",priority:0,members:[],policy:{schedules:[]}}]' "$temp_dir/original-bundle.json" >"$temp_dir/mutated-bundle.json"
    plan_bundle "$temp_dir/mutated-bundle.json" "$temp_dir/mutated-plan.json"
    apply_bundle "$temp_dir/mutated-bundle.json" "$temp_dir/mutated-plan.json" "$temp_dir/mutated-apply.json"
    jq -e '.result.changed == true and .result.error == null' "$temp_dir/mutated-apply.json" >/dev/null
    mutated=1
    api_get /policy-bundle/ >"$temp_dir/mutated-export.json"
    jq -e 'any(.result.bundle.groups[]; .id == "m63-smoke-temp")' "$temp_dir/mutated-export.json" >/dev/null

    restore_original
    api_get /policy-bundle/ >"$temp_dir/restored-export.json"
    restored_normalized="$(jq -S -c '.result.bundle | del(.exportedAt)' "$temp_dir/restored-export.json")"
    [ "$original_normalized" = "$restored_normalized" ]
    [ "$a_before" = "$(fingerprint "$A_HOST")" ]
    [ "$b_before" = "$(fingerprint "$B_HOST")" ]
    printf 'SUMMARY result=pass mode=device probe=allowed-and-rejected webhook=unchanged import=dry-run-conflict-apply-restore residual_group=none A=read-only B=stable\n'
}

case "$MODE" in
    local) run_local ;;
    device) run_device ;;
    *) echo 'M63_SMOKE_MODE must be local or device' >&2; exit 2 ;;
esac
