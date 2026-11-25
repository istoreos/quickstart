#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
test_root="$(mktemp -d /tmp/qs-ops-XXXXXX)"
server_pid=
trap '[ -z "$server_pid" ] || { kill "$server_pid" 2>/dev/null; wait "$server_pid" 2>/dev/null; }; rm -r -- "$test_root"' EXIT HUP INT TERM

write_binary() {
    cat >"$1" <<'EOF'
#!/bin/sh
if [ "${1-}" = serve ] && [ "${2-}" = --help ]; then
    echo '--http-unix value'
fi
exit 0
EOF
    printf '%s\n' "$2" >>"$1"
    chmod 0755 "$1"
}

prepare_fixture() {
    fixture="$1"
    mkdir -p "$fixture/root/usr/sbin" "$fixture/root/www/luci-static/quickstart/i18n" \
        "$fixture/root/usr/lib/lua/luci/view/quickstart" "$fixture/root/usr/lib/lua/luci/controller" \
        "$fixture/root/var/run/quickstart" "$fixture/root/etc/config" \
        "$fixture/input/luci-static/quickstart/i18n" "$fixture/backups" "$fixture/stage"
    write_binary "$fixture/root/usr/sbin/quickstart" old-binary
    cat >"$fixture/root/usr/lib/lua/luci/controller/istore_backend.lua" <<'EOF'
local sock = nixio.socket("unix", "stream")
sock:connect("/var/run/quickstart/http.sock")
EOF
    printf 'old-index\n' >"$fixture/root/www/luci-static/quickstart/index.js"
    printf '{"en":{"old":"Old"}}\n' >"$fixture/root/www/luci-static/quickstart/i18n/en.json"
    printf 'local asset_version = "old"\n' >"$fixture/root/usr/lib/lua/luci/view/quickstart/main.htm"
    for config_name in network dhcp firewall eqos floatip; do
        printf '%s-config\n' "$config_name" >"$fixture/root/etc/config/$config_name"
    done
    write_binary "$fixture/new-binary" new-binary
    printf 'new-index\n' >"$fixture/input/luci-static/quickstart/index.js"
    printf 'new-css\n' >"$fixture/input/luci-static/quickstart/style.css"
    printf 'new-vendor\n' >"$fixture/input/luci-static/quickstart/vendor.js"
    printf '{"en":{"new":"New"}}\n' >"$fixture/input/luci-static/quickstart/i18n/en.json"
    (cd "$fixture/input" && zip -qr "$fixture/web.zip" luci-static)
    cat >"$fixture/service" <<'EOF'
#!/bin/sh
state="${QUICKSTART_TEST_SERVICE_STATE:?}"
printf '%s\n' "$1" >>"${state}.events"
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

for mismatch in lua backend; do
    fixture="$test_root/old-$mismatch"
    prepare_fixture "$fixture"
    if [ "$mismatch" = lua ]; then
        printf 'nixio.connect("127.0.0.1", 3038)\n' >"$fixture/root/usr/lib/lua/luci/controller/istore_backend.lua"
    else
        printf '#!/bin/sh\necho "--addr value"\n' >"$fixture/root/usr/sbin/quickstart"
    fi
    if run_installer "$fixture" mismatch - 1 >"$fixture/error" 2>&1; then
        echo "installer accepted legacy $mismatch" >&2; exit 1
    fi
    grep -Fq 'migrat' "$fixture/error"
    grep -Fqx 'old-index' "$fixture/root/www/luci-static/quickstart/index.js"
    [ "$(cat "$fixture/service.state")" = running ]
    [ ! -e "$fixture/service.state.events" ]
    [ ! -e "$fixture/backups/mismatch" ]
done
echo 'Legacy Lua/backend rejected before stopping service or replacing files'

unix_fixture="$test_root/unix"
prepare_fixture "$unix_fixture"
python3 - "$unix_fixture/root/var/run/quickstart/http.sock" "$unix_fixture/request" <<'PY' &
import http.server
from pathlib import Path
import socketserver
import sys

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        Path(sys.argv[2]).write_text(self.path)
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'{"success":-1001}')
    def log_message(self, *args):
        pass

with socketserver.UnixStreamServer(sys.argv[1], Handler) as server:
    server.serve_forever()
PY
server_pid=$!
for attempt in 1 2 3 4 5 6 7 8 9 10; do
    [ -S "$unix_fixture/root/var/run/quickstart/http.sock" ] && break
    sleep 0.1
done
run_installer "$unix_fixture" unix http://127.0.0.1:1/cgi-bin/luci/istore/system/version/ 1 >/dev/null
grep -Fxq '/cgi-bin/luci/istore/system/version/' "$unix_fixture/request"
grep -Fqx 'new-binary' "$unix_fixture/root/usr/sbin/quickstart"
kill "$server_pid"
wait "$server_pid" 2>/dev/null || true
server_pid=
echo 'Health check uses the Unix socket even when the URL TCP port is unavailable'

# Execute the real rollback command against the fixture instead of SSH or the
# host filesystem, including its transport preflight before stopping services.
mkdir -p "$unix_fixture/mockbin" "$unix_fixture/deployments" "$unix_fixture/servers"
printf 'SERVER_REF=fixture\n' >"$unix_fixture/deployments/fixture.env"
printf 'SSH_TARGET=fixture\n' >"$unix_fixture/servers/fixture.env"
cat >"$unix_fixture/mockbin/ssh" <<'PY'
#!/usr/bin/env python3
import os
import subprocess
import sys
from pathlib import Path

root = Path(os.environ['QUICKSTART_TEST_FIXTURE']) / 'root'
command = sys.argv[-1]
for path in ('/usr/lib/lua/luci/controller/istore_backend.lua', '/lib/upgrade/keep.d/quickstart-state'):
    command = command.replace(path, str(root) + path)
command = command.replace('pidof quickstart >/dev/null', 'true')
sys.exit(subprocess.run(['sh', '-c', command]).returncode)
PY
chmod 0755 "$unix_fixture/mockbin/ssh"
run_rollback() {
    QUICKSTART_TEST_FIXTURE="$unix_fixture" QUICKSTART_TEST_SERVICE_STATE="$unix_fixture/service.state" \
    PATH="$unix_fixture/mockbin:$PATH" ENV_DIR="$unix_fixture/env" DEPLOY_TARGET=fixture \
    DEPLOYMENT_META_DIR="$unix_fixture/deployments" SERVER_META_DIR="$unix_fixture/servers" \
    REMOTE_BACKUP_DIR="$unix_fixture/backups" ROLLBACK_RELEASE=unix \
    REMOTE_BINARY="$unix_fixture/root/usr/sbin/quickstart" REMOTE_SERVICE="$unix_fixture/service" \
    REMOTE_WEB_DIR="$unix_fixture/root/www/luci-static/quickstart" \
    REMOTE_TEMPLATE="$unix_fixture/root/usr/lib/lua/luci/view/quickstart/main.htm" \
        "$SCRIPT_DIR/rollback.sh"
}
run_rollback >/dev/null
grep -Fqx 'old-binary' "$unix_fixture/root/usr/sbin/quickstart"
grep -Fqx 'old-index' "$unix_fixture/root/www/luci-static/quickstart/index.js"
cp "$unix_fixture/service.state.events" "$unix_fixture/events.before"
printf '#!/bin/sh\necho "--addr value"\n' >"$unix_fixture/backups/unix/quickstart"
if run_rollback >"$unix_fixture/rollback-error" 2>&1; then
    echo 'rollback accepted a legacy TCP backend' >&2; exit 1
fi
grep -Fq 'restore legacy TCP pairs manually' "$unix_fixture/rollback-error"
cmp "$unix_fixture/events.before" "$unix_fixture/service.state.events"
grep -Fqx 'old-binary' "$unix_fixture/root/usr/sbin/quickstart"
echo 'Manual rollback restores compatible backups and rejects TCP backups before stopping service'
