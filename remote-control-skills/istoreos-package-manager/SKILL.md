---
name: istoreos-package-manager
description: iStoreOS 软件/插件的搜索与安装闭环（iStore meta / opkg / Docker 应用分流；支持选路径 autoconf；失败时采集证据并验证结果）。
---

# iStoreOS Package Manager

当用户想“搜索/安装/升级/卸载”软件，或出现“安装/下载失败”时，按下面流程执行。

## 0) 禁忌与前置

- 目标系统是 iStoreOS（OpenWrt 风格），只使用 `is-opkg/opkg`；不要建议 `apt/yum/dnf/apk/pacman/brew/systemctl`。
- 先分流：iStore meta / opkg 包 / Docker 类 iStore 应用（istorec）。

## 1) 分流（必须先做）

1. **iStore meta 应用优先**：如果存在 meta（`/usr/lib/opkg/meta/<name>.json`），安装包名为 `app-meta-<name>`。
2. **Docker 类 iStore 应用**：meta 的 `depends` 含 `docker-deps` 或已安装后存在 `/usr/libexec/istorec/<name>.sh`。
3. **否则当作 opkg 包**：走 `opkg list/info/install`。

## 2) 搜索（按分流走）

- iStore meta：
  - 先把用户“口头名”映射到 meta `name`（可能需要反问确认）。
  - 当用户只给“口头描述/功能点”或出现多候选时：优先用 `istoreos-app-search` 输出 Top3，再让用户回复 `name` 确认（未确认前不要安装）：
    - `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-app-search/scripts/search.sh" "<keyword>" 3`
    - 固定展示格式：
      1) `<name>`: `<title>`
         - type: `<type_hint>`
         - entry: `<entry>`
         - why: `<why>` score=`<score>`
      反问：请回复上面条目的 `name` 确认要安装哪一个。
  - 若 Top3 为空/都不对：不要停止，改走降级分支并反问用户选哪条路：
    1) 继续当作 OpenWrt 包：走 opkg 搜索（见下方 `opkg`）
    2) 如果用户其实在找 Docker 镜像：走 Docker 镜像搜索（见下方 `Docker 镜像`）
- opkg：
  - 默认不要先跑 `opkg update`，因为它会写包列表并占用小根系统/overlay 空间；先用现有索引 `opkg list | grep -i <kw>` 和 `opkg info <pkg>`。
  - 只有用户确认需要刷新索引后，才设置 `CONFIRM_OPKG_UPDATE=YES` 调用脚本或手动执行 `opkg update`。
  - 推荐用脚本（更一致）：
    - 名称/描述匹配：`SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-package-manager/scripts/opkg-find.sh" "<regexp>" 30`
    - 包详情：`SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-package-manager/scripts/opkg-info.sh" "<pkg|regexp>" 8`
    - 文件归属包：`SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-package-manager/scripts/opkg-search-file.sh" "/path/to/file" 30`
- Docker 镜像（可选）：
  - 先确保 kspeeder/istoreenhance 可用（否则先安装/启用）：转 `istoreos-docker-acceleration-istoreenhance`
  - 若启用 `istoreenhance` 且 registry 可用：用 `registry-search.sh <kw>`（默认查询 `https://registry.linkease.net:5443/v1/search`，可用环境变量 `ISTORE_REGISTRY_SEARCH_URL` 覆盖）
  - registry 不可用时不要硬搜：让用户提供更精确镜像名或转 iStore meta 搜索。

## 3) 安装/升级/卸载

本节所有安装、升级、卸载、autoconf、服务启停都是写操作。必须先向用户说明将执行的命令、目标包/服务、空间风险和回滚方式；用户确认后才执行。

- iStore meta（推荐）：
  - 确认后安装：`is-opkg install app-meta-<name>`
  - 确认后升级：`is-opkg upgrade app-meta-<name>`
  - 确认后卸载：`is-opkg remove app-meta-<name>`
  - 若 meta 支持 `autoconf`（如 `path/enable`）：确认 base path 不在小根系统/overlay 后，才走 `AUTOCONF=<name> path=<base> enable=<0|1>` 的自动配置路径。
  - 安装后若“已安装但不生效/服务没启动/插件未启用”：转 `istoreos-service-manager`，优先执行：
    - `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; CONFIRM_SERVICE_APPLY=YES sh "$SKILLS_DIR/istoreos-service-manager/scripts/ensure.sh" <name> <name>`（通常 service/uci 同名；不确定就先 `ls -la /etc/init.d/<name>` 与 `test -f /etc/config/<name>` 证据确认）
  - 若服务名不确定（更通用）：先用 `opkg files <pkg>` 定位 init 脚本，再一键确保：
    - `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; CONFIRM_SERVICE_APPLY=YES sh "$SKILLS_DIR/istoreos-package-manager/scripts/ensure_services_from_pkg.sh" <pkg>`
- opkg 包：
  - 确认后执行：`is-opkg install <pkg>`（没有 is-opkg 才用 `opkg install <pkg>`）
- Docker 类 iStore 应用（istorec）：
  - 先确保 Docker 可用（必要时转 `istoreos-docker-basics`）。
  - 安装/升级容器应用前先做空间检查（避免把系统盘/overlay 打满）：
    - `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-docker-basics/scripts/check_space.sh"`
  - 数据目录/安装路径优先复用应用自带路径算法 + is-opkg autoconf（不要手写 `config_path` 规则）。

## 4) 验证闭环（必须做）

- meta/opkg：`opkg status <pkg>` 或 `opkg list-installed | grep -F <pkg>`
- 有服务的：`/etc/init.d/<svc> status`；需要 `restart` 时必须先确认并优先转 `istoreos-service-manager`
- Docker 类：`/usr/libexec/istorec/<name>.sh status` 或 `docker ps --all -f 'name=^/<name>$'`

## 5) 失败时（不要猜）

转 `istoreos-logs-and-diagnostics`，先收集最小证据再定位：
- 安装/下载输出、`logread` 片段、`df -h`（空间相关）、Docker 相关则加 `docker version`/`uci show dockerd`。
