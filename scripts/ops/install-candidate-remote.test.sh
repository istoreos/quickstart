#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
test_root="$(mktemp -d)"
trap 'rm -r -- "$test_root"' EXIT HUP INT TERM

grep -Fq 'command -v uclient-fetch' "$SCRIPT_DIR/install-candidate-remote.sh"
grep -Fq -- "--header='X-Forwarded-Sid: deploy-smoke'" "$SCRIPT_DIR/install-candidate-remote.sh"

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
    printf 'new-css\n' >"$fixture/input/luci-static/quickstart/style.css"
    printf 'new-vendor\n' >"$fixture/input/luci-static/quickstart/vendor.js"
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
    style_sha="${QUICKSTART_TEST_STYLE_SHA:-$(sha256sum "$fixture/input/luci-static/quickstart/style.css" | cut -d ' ' -f1)}"
    vendor_sha="$(sha256sum "$fixture/input/luci-static/quickstart/vendor.js" | cut -d ' ' -f1)"
    QUICKSTART_TEST_SERVICE_STATE="$fixture/service.state" \
        "$SCRIPT_DIR/install-candidate-remote.sh" \
        "$release" "$fixture/new-binary" "$fixture/web.zip" \
        "$fixture/root/usr/sbin/quickstart" "$fixture/root/www/luci-static/quickstart" \
        "$fixture/root/usr/lib/lua/luci/view/quickstart/main.htm" "$fixture/backups" \
        "$fixture/service" "$binary_sha" "$index_sha" "$en_sha" "0.14.0-test" \
        "$health" "$fixture/stage" "$attempts" "$fixture/root" "$style_sha" "$vendor_sha"
}

success_fixture="$test_root/success"
prepare_fixture "$success_fixture"
run_installer "$success_fixture" success - 1 >/dev/null
grep -Fqx 'new-binary' "$success_fixture/root/usr/sbin/quickstart"
grep -Fqx 'new-index' "$success_fixture/root/www/luci-static/quickstart/index.js"
grep -Fq 'local asset_version = "0.14.0-test"' "$success_fixture/root/usr/lib/lua/luci/view/quickstart/main.htm"
grep -Fqx 'old-binary' "$success_fixture/backups/success/quickstart"
cmp "$success_fixture/backups/success/config-sha.before" "$success_fixture/backups/success/config-sha.after"
grep -Fxq '/etc/quickstart/' "$success_fixture/root/lib/upgrade/keep.d/quickstart-state"

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
[ ! -e "$rollback_fixture/root/lib/upgrade/keep.d/quickstart-state" ]

keep_fixture="$test_root/keep-rollback"
prepare_fixture "$keep_fixture"
mkdir -p "$keep_fixture/root/lib/upgrade/keep.d"
printf '/etc/quickstart/custom.json\n' >"$keep_fixture/root/lib/upgrade/keep.d/quickstart-state"
if run_installer "$keep_fixture" keep-rollback http://127.0.0.1:1 1 >/dev/null 2>&1; then
    echo 'unhealthy candidate accepted' >&2; exit 1
fi
grep -Fxq '/etc/quickstart/custom.json' "$keep_fixture/root/lib/upgrade/keep.d/quickstart-state"


legacy_fixture="$test_root/legacy"
prepare_fixture "$legacy_fixture"
cat >"$legacy_fixture/root/usr/lib/lua/luci/view/quickstart/main.htm" <<'EOF'
<%+header%>
local jsonc = require "luci.jsonc"
window.vue_lang_data = '/luci-static/quickstart/i18n/<%=lang%>.json?v=0.8.17-r1';
<script type="module" src="/luci-static/quickstart/index.js?v=0.8.17-r1"></script>
<link href="/luci-static/quickstart/style.css?v=0.8.17-r1">
<%+footer%>
EOF
run_installer "$legacy_fixture" legacy - 1 >/dev/null
grep -Fq 'local asset_version = "0.14.0-test"' "$legacy_fixture/root/usr/lib/lua/luci/view/quickstart/main.htm"
[ "$(grep -c '?v=<%=asset_version%>' "$legacy_fixture/root/usr/lib/lua/luci/view/quickstart/main.htm")" = 3 ]
grep -Fq '<%+header%>' "$legacy_fixture/root/usr/lib/lua/luci/view/quickstart/main.htm"
grep -Fq '?v=0.8.17-r1' "$legacy_fixture/backups/legacy/main.htm"
echo 'candidate installer modern/legacy success and rollback ok'

css_fixture="$test_root/css-mismatch"
prepare_fixture "$css_fixture"
if QUICKSTART_TEST_STYLE_SHA=incorrect run_installer "$css_fixture" css-mismatch - 1 >/dev/null 2>&1; then
    echo 'installer accepted mismatched CSS' >&2; exit 1
fi
grep -Fqx 'old-binary' "$css_fixture/root/usr/sbin/quickstart"
echo 'CSS mismatch rejected before changing installed candidate'

indented_fixture="$test_root/indented"
prepare_fixture "$indented_fixture"
cat >"$indented_fixture/root/usr/lib/lua/luci/view/quickstart/main.htm" <<'EOF'
<%+header%>
    local jsonc = require "luci.jsonc"
    local asset_version = "old"
<%+footer%>
EOF
run_installer "$indented_fixture" indented - 1 >/dev/null
[ "$(grep -c 'local asset_version =' "$indented_fixture/root/usr/lib/lua/luci/view/quickstart/main.htm")" = 1 ]
grep -Fqx '    local asset_version = "0.14.0-test"' "$indented_fixture/root/usr/lib/lua/luci/view/quickstart/main.htm"
echo 'Indented cache key updated without a shadowing declaration'
