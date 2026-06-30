---
name: istoreos-docker-basics
description: iStoreOS Docker 可用性闭环（检测/安装/启停/验证/基础排错）；涉及镜像拉取/构建时，先确保 kspeeder/istoreenhance 已安装启用并生效（自动走加速闭环）。
---

# iStoreOS Docker Basics

目标：确保 `docker` + `dockerd` 在 iStoreOS 上可用。

## 流程

1) 检测安装：`command -v docker; command -v dockerd`
2) 检测服务：`/etc/init.d/dockerd status || ps w | grep -E '[d]ockerd'`
3) 安装/修复（写操作，必须先确认；走 iStoreOS 路径）：优先 iStore meta，其次 `is-opkg/opkg`（不要 apt/yum/systemctl）
4) 启用启动是写操作，必须先确认；转 `istoreos-service-manager` 使用 `CONFIRM_SERVICE_APPLY=YES` 的脚本门槛
5) 验证闭环（至少 2 条）：`docker version`（含 Server）/ `docker info` / `docker ps`
6) 安装/升级 Docker 类应用前的空间检查（关键）
   - `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-docker-basics/scripts/check_space.sh"`
   - 若输出 `need-confirmation:`（例如 DockerRootDir 可用空间 < 2GiB，或 data_root 在 `/overlay/upper/opt/docker`），先处理空间/迁移 data_root 再继续安装镜像/容器
   - 迁移 data_root 属于危险写操作：转 `istoreos-docker-data-root-migrate`（必须明确征得用户同意，并先备份再改）
7) 任何可能拉镜像的动作（`docker pull/build/run`）前置：先跑加速闭环
   - `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-docker-acceleration-istoreenhance/scripts/ensure_ready.sh"`
   - 若输出 `need-confirmation:`（例如未安装或 `cache` 未配置），先按提示完成并等待用户确认，再继续 Docker 动作
8) 镜像拉取失败（超时/连接重置/TLS 握手超时等）：
   - 不要直接进入“换源/代理/证书”泛化建议；优先回到第 6 步，确保 `istoreenhance/kspeeder` 真正生效
9) 仍失败时转 `istoreos-logs-and-diagnostics` 收集 `logread` + `uci show dockerd` + `uci show istoreenhance`
