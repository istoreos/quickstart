#!/bin/sh
set -eu

# This script runs on the OpenWrt target. Keep it dependency-light and pass
# every deployment-specific value as a positional argument.
[ "$#" -eq 16 ] || [ "$#" -eq 18 ] || {
    echo "usage: $0 RELEASE_ID BINARY_TMP WEB_ZIP_TMP BINARY WEB_DIR TEMPLATE BACKUP_ROOT SERVICE BINARY_SHA INDEX_SHA EN_SHA ASSET_VERSION HEALTH_URL STAGE_ROOT HEALTH_ATTEMPTS CONFIG_ROOT" >&2
    exit 2
}

release_id="$1"
binary_tmp="$2"
web_zip_tmp="$3"
remote_binary="$4"
remote_web_dir="$5"
remote_template="$6"
backup_root="$7"
remote_service="$8"
expected_binary_sha="$9"
shift 9
expected_index_sha="$1"
expected_en_sha="$2"
asset_version="$3"
health_url="$4"
stage_root="$5"
health_attempts="$6"
config_root="$7"
keep_path="${config_root%/}/lib/upgrade/keep.d/quickstart-state"
expected_style_sha="${8:-}"
expected_vendor_sha="${9:-}"

case "$release_id:$asset_version" in
    *[!A-Za-z0-9._:-]*) echo "invalid release or asset version" >&2; exit 2 ;;
esac

backup_dir="${backup_root}/${release_id}"
stage_dir="${stage_root}/${release_id}"
stage_web="${stage_dir}/luci-static/quickstart"
next_web="${remote_web_dir}.quickstart-next"
previous_web="${remote_web_dir}.quickstart-previous"
template_next="${remote_template}.quickstart-next"
rollback_active=0
install_complete=0

health_probe() {
    probe_url="$1"
    if command -v curl >/dev/null 2>&1; then
        curl -fsS --max-time 3 -H 'X-Forwarded-Sid: deploy-smoke' "$probe_url" >/dev/null 2>&1
    elif command -v uclient-fetch >/dev/null 2>&1; then
        uclient-fetch --quiet --timeout=3 --header='X-Forwarded-Sid: deploy-smoke' -O /dev/null "$probe_url" >/dev/null 2>&1
    elif command -v wget >/dev/null 2>&1; then
        wget -q -T 3 -O /dev/null --header='X-Forwarded-Sid: deploy-smoke' "$probe_url" >/dev/null 2>&1
    else
        echo 'no supported HTTP client for candidate health check' >&2
        return 127
    fi
}

restore_release() {
    rollback_status="$?"
    trap - EXIT HUP INT TERM
    if [ "$rollback_active" = 1 ] && [ "$install_complete" != 1 ]; then
        echo "candidate install failed; restoring ${backup_dir}" >&2
        "$remote_service" stop >/dev/null 2>&1 || true
        cp -p "${backup_dir}/quickstart" "$remote_binary" || true
        chmod 0755 "$remote_binary" 2>/dev/null || true
        rm -rf "$remote_web_dir"
        cp -a "${backup_dir}/web" "$remote_web_dir" || true
        cp -p "${backup_dir}/main.htm" "$remote_template" || true
        if [ -f "${backup_dir}/quickstart-state.keep" ]; then
            cp -p "${backup_dir}/quickstart-state.keep" "$keep_path" || true
        elif [ -f "${backup_dir}/quickstart-state.keep.absent" ]; then
            rm -f "$keep_path"
        fi
        "$remote_service" start >/dev/null 2>&1 || true
    fi
    rm -rf "$next_web" "$previous_web" "$template_next" "$stage_dir"
    exit "$rollback_status"
}
trap restore_release EXIT HUP INT TERM

command -v sha256sum >/dev/null
command -v unzip >/dev/null
command -v awk >/dev/null
test -f "$binary_tmp"
test -f "$web_zip_tmp"
test -f "$remote_template"
test -d "$remote_web_dir"
test -x "$remote_service"
[ ! -e "$backup_dir" ] || {
    echo "release backup already exists: ${backup_dir}" >&2
    exit 1
}

rm -rf "$stage_dir" "$next_web" "$previous_web" "$template_next"
mkdir -p "$stage_dir" "$backup_dir"
unzip -q "$web_zip_tmp" -d "$stage_dir"
test -f "${stage_web}/index.js"
test -f "${stage_web}/i18n/en.json"

actual_binary_sha="$(sha256sum "$binary_tmp" | cut -d ' ' -f1)"
actual_index_sha="$(sha256sum "${stage_web}/index.js" | cut -d ' ' -f1)"
actual_en_sha="$(sha256sum "${stage_web}/i18n/en.json" | cut -d ' ' -f1)"
[ "$actual_binary_sha" = "$expected_binary_sha" ]
[ "$actual_index_sha" = "$expected_index_sha" ]
[ "$actual_en_sha" = "$expected_en_sha" ]
[ -z "$expected_style_sha" ] || [ "$(sha256sum "${stage_web}/style.css" | cut -d ' ' -f1)" = "$expected_style_sha" ]
[ -z "$expected_vendor_sha" ] || [ "$(sha256sum "${stage_web}/vendor.js" | cut -d ' ' -f1)" = "$expected_vendor_sha" ]

awk -v version="$asset_version" '
    {
        lines[NR] = $0
        if ($0 ~ /^[[:space:]]*local asset_version = /) known = 1
        if ($0 ~ /local jsonc = require/) anchor = NR
    }
    END {
        if (!known && !anchor) exit 42
        for (i = 1; i <= NR; i++) {
            line = lines[i]
            if (line ~ /^[[:space:]]*local asset_version = /) {
                sub(/local asset_version = .*/, "local asset_version = \"" version "\"", line)
                print line
                continue
            }
            # The pre-versioned LuCI template has exactly these three assets.
            # Keep its system header/footer; only retire their fixed cache keys.
            if (!known && line ~ /\/luci-static\/quickstart\//)
                changed += gsub(/\?v=[A-Za-z0-9._-]+/, "?v=<%=asset_version%>", line)
            print line
            if (!known && i == anchor)
                print "local asset_version = \"" version "\""
        }
        if (!known && changed != 3) exit 42
    }
' "$remote_template" >"$template_next"
chmod 0644 "$template_next"

cp -p "$remote_binary" "${backup_dir}/quickstart"
cp -a "$remote_web_dir" "${backup_dir}/web"
cp -p "$remote_template" "${backup_dir}/main.htm"
if [ -f "$keep_path" ]; then
    cp -p "$keep_path" "${backup_dir}/quickstart-state.keep"
else
    : >"${backup_dir}/quickstart-state.keep.absent"
fi
{
    for config_name in network dhcp firewall eqos floatip; do
        config_path="${config_root%/}/etc/config/${config_name}"
        [ -f "$config_path" ] && sha256sum "$config_path"
    done
} >"${backup_dir}/config-sha.before"
opkg list-installed quickstart luci-app-quickstart luci-app-eqos >"${backup_dir}/packages.before" 2>/dev/null || true

mv "$stage_web" "$next_web"
rollback_active=1
"$remote_service" stop
cp "$binary_tmp" "${remote_binary}.quickstart-next"
chmod 0755 "${remote_binary}.quickstart-next"
mv "${remote_binary}.quickstart-next" "$remote_binary"
mv "$remote_web_dir" "$previous_web"
mv "$next_web" "$remote_web_dir"
mv "$template_next" "$remote_template"
# The binary rollout must preserve the new JSON configuration on a later
# sysupgrade, even when the appliance still has the old package metadata.
mkdir -p "$(dirname "$keep_path")"
printf '# Quickstart persistent configuration and recovery snapshots\n/etc/quickstart/\n' >"$keep_path"
chmod 0644 "$keep_path"
"$remote_service" start

health_ok=0
attempt=1
while [ "$attempt" -le "$health_attempts" ]; do
    if "$remote_service" status >/dev/null 2>&1; then
        if [ "$health_url" = "-" ] || health_probe "$health_url"; then
            health_ok=1
            break
        fi
    fi
    sleep 1
    attempt=$((attempt + 1))
done
[ "$health_ok" = 1 ]

[ "$(sha256sum "$remote_binary" | cut -d ' ' -f1)" = "$expected_binary_sha" ]
[ "$(sha256sum "${remote_web_dir}/index.js" | cut -d ' ' -f1)" = "$expected_index_sha" ]
[ "$(sha256sum "${remote_web_dir}/i18n/en.json" | cut -d ' ' -f1)" = "$expected_en_sha" ]
grep -Fq "local asset_version = \"${asset_version}\"" "$remote_template"

{
    for config_name in network dhcp firewall eqos floatip; do
        config_path="${config_root%/}/etc/config/${config_name}"
        [ -f "$config_path" ] && sha256sum "$config_path"
    done
} >"${backup_dir}/config-sha.after"
cmp "${backup_dir}/config-sha.before" "${backup_dir}/config-sha.after"

install_complete=1
rollback_active=0
rm -rf "$previous_web" "$stage_dir"
rm -f "$binary_tmp" "$web_zip_tmp"
trap - EXIT HUP INT TERM

printf 'backup=%s\n' "$backup_dir"
printf 'installed_binary_sha=%s\n' "$expected_binary_sha"
printf 'installed_index_sha=%s\n' "$expected_index_sha"
printf 'installed_en_sha=%s\n' "$expected_en_sha"
printf 'asset_version=%s\n' "$asset_version"
printf 'deploy ok\n'
