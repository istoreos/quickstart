---
name: istoreos-docker-acceleration-istoreenhance
description: Docker 镜像下载慢/失败时的 iStoreOS 加速闭环（istoreenhance/kspeeder：安装→autoconf→启用→registry_mirrors→search API 验证→降级策略）。
---

# Docker Acceleration via iStoreEnhance (kspeeder)

当用户进行镜像拉取/构建或出现“下载慢/失败”，先确保加速链路可用，再继续 Docker 动作。

## 流程

0) 前置：Docker 必须可用（否则先转 `istoreos-docker-basics`）

1) 一键准备（推荐）

- 只读/计划阶段先运行：`SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-docker-acceleration-istoreenhance/scripts/ensure_ready.sh"`
- 若脚本输出 `need-confirmation:`，向用户说明将执行的安装、autoconf、UCI commit、服务启停影响。
- 用户明确确认后才设置确认门槛执行：`SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; CONFIRM_ISTOREENHANCE_APPLY=YES sh "$SKILLS_DIR/istoreos-docker-acceleration-istoreenhance/scripts/ensure_ready.sh"`
- 该脚本会优先尝试自动选择 `base path` 并执行 autoconf（复用 `istoreos-storage-path` 的选盘策略）；若无法自动选盘，不会回退系统盘，需要先确认路径。

1) 检测是否安装：
- `opkg status app-meta-istoreenhance || opkg status istoreenhance || test -x /etc/init.d/istoreenhance`

2) 未安装：引导安装；说明会写包列表/安装包并占用空间，等待用户确认后执行
- 确认后执行：`is-opkg install app-meta-istoreenhance`
- 若需要装到硬盘：用 `istoreos-storage-path` 先选 `base path`，然后走 autoconf（`path/enable`）

3) 配置检查与修复：
- `uci -q show istoreenhance`（`enabled=1`，`cache` 非空）
- `cache` 为空时优先用 autoconf 修复（不要手改猜路径）
- 注意：某些情况下安装后 `enabled` 可能仍为 `0`（服务会拒绝启动）；必须把 `istoreenhance.*.enabled` 设为 `1` 后再启动服务（或重新跑 autoconf）。

4) 确保服务运行：
- 使用脚本门槛：`CONFIRM_ISTOREENHANCE_APPLY=YES sh "$SKILLS_DIR/istoreos-docker-acceleration-istoreenhance/scripts/ensure_ready.sh"`
- 不要绕过脚本直接启停服务；如果必须手动执行 `/etc/init.d/istoreenhance enable/restart`，先逐条向用户说明影响并等待确认。

5) 验证联动：
- `uci -q show dockerd | grep -F registry_mirrors || true`（常见加速源示例：`https://registry.linkease.net:5443`；以实际配置为准）
-（需要搜索镜像时）探测（按实际加速源域名/地址）：`nslookup <registry-host>` + `curl -fsS '<registry-search-url>?n=1' --data-urlencode 'q=busybox'`

6) 降级策略：
- 加速链路不可用时可以尝试 `docker pull <image>`，但要明确告知“可能不走加速”；需要搜索/推荐时不要胡猜。
