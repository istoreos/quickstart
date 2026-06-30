---
name: quickstart-router-api
description: Safely inspect and control iStoreOS/OpenWrt QuickStart router backend APIs through LuCI. Use when Codex needs to connect to a router by IP/host with a LuCI username and password, call /cgi-bin/luci/istore QuickStart APIs, read router status, inspect network/Wi-Fi/NAS/RAID/SMART/share/DDNS/Docker state, or perform confirmed configuration changes using the bundled Go helper, mise-provided Go fallback, or curl fallback.
---

# QuickStart Router API

## Operating Model

Use this skill to access a QuickStart router backend through LuCI, not by calling the backend port directly. The deployed service normally runs `quickstart serve` on `127.0.0.1:3038`; LuCI exposes it through `/cgi-bin/luci/istore` and forwards the authenticated session as `X-Forwarded-Sid`.

Require the user to provide the router host/IP, LuCI username, and LuCI password at runtime. Keep credentials in memory only for the current command process, and pass the password with `--password-env ROUTER_PASSWORD` rather than putting the secret in process arguments. Never store credentials in this skill, repository files, shell history, logs, or generated config files.

## Safety Rules

Default to read-only inspection. Before any write operation, state the exact endpoint, request body, and expected device effect, then ask for confirmation. Treat these as dangerous and require explicit confirmation: reboot, poweroff, password change, network interface writes, Wi-Fi changes, disk init/format/mount, NAS sandbox commit/reset, RAID create/delete/add/remove/recover, app install, and LAN control write operations.

After a write operation, read back the relevant status endpoint and report the result. If an API response has `success != 0`, treat it as a failure even when HTTP status is 200.

## Preferred Tool

Default execution order is: local Go, mise-provided Go, then curl fallback. Choose the runtime in this order:

1. If `go` is available, run `go run`.
2. If `go` is unavailable but `mise` is available, run through `mise exec go@latest --`.
3. If neither Go nor mise is available, use the curl fallback.

Check the local environment:

```bash
command -v go || command -v mise || command -v curl
```

With local Go:

```bash
read -r -s -p "LuCI password: " ROUTER_PASSWORD; printf '\n'
export ROUTER_PASSWORD
go run scripts/qsctl.go --host 192.168.30.244 --user root --password-env ROUTER_PASSWORD get /system/status/
unset ROUTER_PASSWORD
```

With mise-provided Go:

```bash
read -r -s -p "LuCI password: " ROUTER_PASSWORD; printf '\n'
export ROUTER_PASSWORD
mise exec go@latest -- go run scripts/qsctl.go --host 192.168.30.244 --user root --password-env ROUTER_PASSWORD get /system/status/
unset ROUTER_PASSWORD
```

For ordinary confirmed POST:

```bash
go run scripts/qsctl.go --host 192.168.30.244 --user root --password-env ROUTER_PASSWORD --confirm-write post /system/auto-check-update/ '{"enable":true}'
```

The helper:

- logs in to `/cgi-bin/luci/`
- keeps LuCI credentials and cookies in memory only
- calls `/cgi-bin/luci/istore/<path>`
- validates JSON request bodies
- blocks every POST unless `--confirm-write` or `CONFIRM_QSCTL_WRITE=YES` is set after user confirmation
- blocks dangerous POST paths unless `--confirm-danger` or `CONFIRM_QSCTL_DANGER=YES` is also set after explicit user confirmation
- returns non-zero on HTTP errors or QuickStart envelope errors
- prints the raw JSON response to stdout

For dangerous confirmed POST, such as reboot:

```bash
go run scripts/qsctl.go --host 192.168.30.244 --user root --password-env ROUTER_PASSWORD --confirm-write --confirm-danger post /system/reboot/ '{}'
```

Use `--host https://192.168.30.244 --insecure` only when the router uses HTTPS with an untrusted local certificate.

Build static binaries only when needed:

```bash
sh scripts/build-qsctl.sh
```

If `go` is unavailable and `mise` is available, build through mise without writing project config:

```bash
mise exec go@latest -- sh scripts/build-qsctl.sh
```

The build script writes artifacts under `quickstart-router-api/bin/`, which is ignored by this skill. Do not commit generated binaries.

## Curl Fallback

Use curl only if Go is unavailable. Curl fallback is for read operations by default; it does not provide the Go helper's write/danger gates. Keep the cookie file in a temporary path and remove it after use. Do not use `set -x`.

```bash
HOST="192.168.30.244"
USER="root"
COOKIE="$(mktemp)"

curl -fsS -c "$COOKIE" -K - >/dev/null <<EOF
url = "http://$HOST/cgi-bin/luci/"
header = "Content-Type: application/x-www-form-urlencoded"
data-urlencode = "luci_username=$USER"
data-urlencode = "luci_password=$ROUTER_PASSWORD"
EOF

curl -fsS -b "$COOKIE" \
  "http://$HOST/cgi-bin/luci/istore/system/status/"

rm -f "$COOKIE"
```

For POST, prefer the Go helper. If curl is the only available tool, require the same user confirmation first and keep the endpoint/body in the transcript:

```bash
curl -fsS -b "$COOKIE" \
  -H "Content-Type: application/json;charset=utf-8" \
  -X POST \
  -d '{"enable":true}' \
  "http://$HOST/cgi-bin/luci/istore/system/auto-check-update/"
```

When using curl, parse the JSON response and check `success`. Do not assume HTTP 200 means success.

## Endpoint Reference

Read `references/api-summary.md` before choosing endpoints. Use paths relative to `/cgi-bin/luci/istore` with the Go helper, for example `/network/status/` rather than the full LuCI path.

Request and response schemas come from the QuickStart swagger YAML files in the upstream project's `backend/yamls` directory. If the live router behaves differently, prefer the live router's response and report the mismatch.
