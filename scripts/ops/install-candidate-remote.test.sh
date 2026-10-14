#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
test_root="$(mktemp -d)"
trap 'rm -r -- "$test_root"' EXIT HUP INT TERM

prepare_fixture() {
    fixture="$1"
    mkdir -p "$fixture/root/usr/sbin" "$fixture/root/www/luci-static/quickstart/i18n" \
        "$fixture/root/usr/lib/lua/luci/view/quickstart" "$fixture/root/etc/config" \
        "$fixture/input/luci-static/quickstart/i18n" "$fixture/backups" "$fixture/stage"
    printf 'old-binary\n' >"$fixture/root/usr/sbin/quickstart"
    chmod 0755 "$fixture/root/usr/sbin/quickstart"
    printf 'old-index\n' >"$fixture/root/www/luci-static/quickstart/index.js"
    printf '{"en":{"old":"Old"}}\n' >"$fixture/root/www/luci-static/quickstart/i18n/en.json"
    printf 'local asset_version = "old"\n' >"$fixture/root/usr/lib/lua/luci/view/quickstart/main.htm"
    for config_name in network dhcp firewall eqos floatip; do
        printf '%s-config\n' "$config_name" >"$fixture/root/etc/config/$config_name"
    done
    printf 'new-binary\n' >"$fixture/new-binary"
    printf 'new-index\n' >"$fixture/input/luci-static/quickstart/index.js"
    printf '{"en":{"new":"New"}}\n' >"$fixture/input/luci-static/quickstart/i18n/en.json"
    (cd "$fixture/input" && zip -qr "$fixture/web.zip" luci-static)
    cat >"$fixture/service" <<'EOF'
#!/bin/sh
state="${QUICKSTART_TEST_SERVICE_STATE:?}"
case "$1" in
    start|restart) printf running >"$state" ;;
    stop) printf stopped >"$state" ;;
    status) [ "$(cat "$state" 2>/dev/null || true)" = running ] ;;
    *) exit 2 ;;
esac
EOF
    chmod 0755 "$fixture/service"
    printf running >"$fixture/service.state"
}

run_installer() {
    fixture="$1"
    release="$2"
    health="$3"
    attempts="$4"
    binary_sha="$(sha256sum "$fixture/new-binary" | cut -d ' ' -f1)"
    index_sha="$(sha256sum "$fixture/input/luci-static/quickstart/index.js" | cut -d ' ' -f1)"
    en_sha="$(sha256sum "$fixture/input/luci-static/quickstart/i18n/en.json" | cut -d ' ' -f1)"
    QUICKSTART_TEST_SERVICE_STATE="$fixture/service.state" \
        "$SCRIPT_DIR/install-candidate-remote.sh" \
        "$release" "$fixture/new-binary" "$fixture/web.zip" \
        "$fixture/root/usr/sbin/quickstart" "$fixture/root/www/luci-static/quickstart" \
        "$fixture/root/usr/lib/lua/luci/view/quickstart/main.htm" "$fixture/backups" \
        "$fixture/service" "$binary_sha" "$index_sha" "$en_sha" "0.14.0-test" \
        "$health" "$fixture/stage" "$attempts" "$fixture/root"
}

success_fixture="$test_root/success"
prepare_fixture "$success_fixture"
run_installer "$success_fixture" success - 1 >/dev/null
grep -Fqx 'new-binary' "$success_fixture/root/usr/sbin/quickstart"
grep -Fqx 'new-index' "$success_fixture/root/www/luci-static/quickstart/index.js"
grep -Fq 'local asset_version = "0.14.0-test"' "$success_fixture/root/usr/lib/lua/luci/view/quickstart/main.htm"
grep -Fqx 'old-binary' "$success_fixture/backups/success/quickstart"
cmp "$success_fixture/backups/success/config-sha.before" "$success_fixture/backups/success/config-sha.after"

rollback_fixture="$test_root/rollback"
prepare_fixture "$rollback_fixture"
set +e
run_installer "$rollback_fixture" rollback http://127.0.0.1:1 1 >/dev/null 2>&1
rollback_status=$?
set -e
[ "$rollback_status" -ne 0 ]
grep -Fqx 'old-binary' "$rollback_fixture/root/usr/sbin/quickstart"
grep -Fqx 'old-index' "$rollback_fixture/root/www/luci-static/quickstart/index.js"
grep -Fq 'local asset_version = "old"' "$rollback_fixture/root/usr/lib/lua/luci/view/quickstart/main.htm"
[ "$(cat "$rollback_fixture/service.state")" = running ]

echo 'candidate installer success and rollback ok'
