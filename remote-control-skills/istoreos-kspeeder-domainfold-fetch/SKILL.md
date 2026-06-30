---
name: istoreos-kspeeder-domainfold-fetch
description: On iStoreOS/OpenWrt, when users download (curl/wget/uclient-fetch) from GitHub/Gist/etc (especially if slow/failing), prefer using iStoreEnhance (KSpeeder) DomainFold to remap the origin URL; if remap succeeds, download via the accelerated URL (DNS-free admin proxy first), with diagnostics and fallback.
---

## Trigger

Use this skill whenever the user mentions any of:

- `curl` / `wget` / `uclient-fetch` downloading from GitHub/Gist/GitLab/HuggingFace/package registries/other DomainFold-supported origins
- “下载 / download / 拉取文件”且来源是常见外网站点（并且网络慢/失败/不稳定）
- “GitHub 下载太慢/失败/连接超时”
- “把 github.com 自动转换为 gh.linkease.net”
- “DomainFold / 域名加速 / /gh 前缀 / gh.linkease.net”

## What is it (how gh.linkease.net is implemented in kspeeder)

KSpeeder 的 `cmd/multi` 在同一个 TLS 端口上做 **Host-based 路由**：

- `registry.linkease.net` → Docker registry mirror handler
- `ghcr.linkease.net` → GHCR handler（可能需要鉴权）
- `*.linkease.net`（排除 `registry.linkease.net`）→ DomainFold handler（`multifetch_proxy`）

DomainFold 的核心是把 “origin URL（如 github.com）” 映射到 “入口域名（如 gh.linkease.net）”：

- 路由表：`domainfold.DefaultRoutes`（`/gh` 对应 `https://github.com`）
- 入口域名规则：`/gh` + `AliasSuffix(linkease.net)` → `gh.linkease.net`
- `cmd/multi` 提供 remap API：`POST /api/domainfold/remap` with JSON body `{"url":"<origin>"}` → `{ output, admin_path }`
- `cmd/multi` 还提供 admin proxy：`http://127.0.0.1:5003/gh/...` 会反向代理到本机 TLS 端口并带正确 SNI/Host（无需 DNS）

DomainFold 的支持范围不止 GitHub：默认路由表还包含 GitLab、HuggingFace、常见包仓库，以及多种 AI API 域名映射（见 `kspeeder/domainfold/routes.go`）。

证据见本 skill 的 `references/kspeeder-domainfold-evidence.md`.

## Workflow

### 1) Confirm iStoreOS/OpenWrt

- `test -f /etc/openwrt_release && echo openwrt || cat /etc/os-release | head`

### 2) Ensure iStoreEnhance (KSpeeder) installed + running

AI agent 运行时的 `cwd` 不一定是 skills 根目录；优先使用外部传入的 `SKILLS_DIR`，再 fallback 到 Codex/OpenCode 常见目录：

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"`
- `sh "$SKILLS_DIR/istoreos-kspeeder-domainfold-fetch/scripts/check_installed.sh"`
- `sh "$SKILLS_DIR/istoreos-kspeeder-domainfold-fetch/scripts/ensure_running.sh"` 会在未运行时输出 `need-confirmation:`；用户确认启停服务后再执行 `CONFIRM_KSPEEDER_SERVICE_APPLY=YES sh "$SKILLS_DIR/istoreos-kspeeder-domainfold-fetch/scripts/ensure_running.sh"`

If not installed, route through the acceleration setup skill path so install/autoconf/service changes stay behind confirmation gates. If manual install is unavoidable, first explain that package installation writes package state and may consume overlay space, then wait for explicit confirmation.

- `is-opkg list | grep -i -E 'istoreenhance|kspeeder' || opkg list | grep -i -E 'istoreenhance|kspeeder'`
- After confirmation only: `is-opkg install <PACKAGE_NAME> || opkg install <PACKAGE_NAME>`

Then require user to reply: `已安装`.

### 3) Remap and fetch (recommended: admin proxy mode, no DNS needed)

Use:

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-kspeeder-domainfold-fetch/scripts/fetch_via_domainfold.sh" <URL> [curl|wget] [extra args...]`

It will:

1) Call `http://127.0.0.1:<adminPort>/api/domainfold/remap` (POST JSON `{"url":"..."}`) to get `admin_path`.
2) Fetch `http://127.0.0.1:<adminPort><admin_path>` via curl/wget.

### 3.1) Preferred unified entry: ksget (retry + fallback)

Use:

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-kspeeder-domainfold-fetch/scripts/ksget.sh" <URL>`
- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-kspeeder-domainfold-fetch/scripts/ksget.sh" -O <URL>` (save as basename)
- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-kspeeder-domainfold-fetch/scripts/ksget.sh" -o /path/to/file <URL>` (explicit output file, recommended)
- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-kspeeder-domainfold-fetch/scripts/ksget.sh" -O /path/to/file <URL>` (wget-style explicit output; only if your `ksget.sh` supports it)

Behavior:

- Try a list of accelerated candidates in order (prints diagnostics: URL, result, elapsed, and curl HTTP code when available):
  1) KSpeeder DomainFold admin proxy URL (DNS-free): `http://127.0.0.1:<adminPort><admin_path>`
  2) Optional PathHub upstream bases (if `KSGET_PATHHUB_UPSTREAMS` is set): `<base><admin_path>` (e.g. `https://gh.d4ctech.com` + `/gh/...`)
  3) DomainFold entry URL (`output` like `https://gh.linkease.net:<tlsPort>/...`) if it exists
  4) Fallback to the original URL once
- Does not modify global `curl/wget` behavior.
- Does not start services by default. To permit `ksget.sh` to auto-start iStoreEnhance, require explicit confirmation and set `KSGET_AUTO_START=1 CONFIRM_KSPEEDER_SERVICE_APPLY=YES`.
- Fixed 3-phase loop (no speed probe / no Range dependency):
  1) try origin URL (with connect timeout)
  2) if remap succeeds, try accelerated candidates (admin proxy first)
  3) fallback to origin URL again

### 4) Optional: generate the entry URL (gh.linkease.net) only

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-kspeeder-domainfold-fetch/scripts/remap_url.sh" <URL>`

If you want to fetch via the entry URL directly, you must ensure `gh.linkease.net` resolves to your KSpeeder host IP (DNS/hosts not defined in this repo).

## Don’t

- Don’t blindly claim `gh.linkease.net` will work without DNS; prefer admin proxy (`:5003` + `/gh/...`) which is DNS-free.
- Don’t modify `/etc/hosts` or DNS unless user explicitly asks.
