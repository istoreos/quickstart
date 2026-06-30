---
name: istoreos-run-executable
description: iStoreOS 处理 .run 可执行文件（本地或 URL）：先获取（可选，支持 KSpeeder 加速下载）、再只读检查（大小/类型/校验和/架构/空间），再强制备份提示与确认，最后用 is-opkg dotrun（可记录/可回滚）或直接 chmod+执行（可带参数）并留存日志。
---

# iStoreOS `.run` Executable Handler

当用户拿到一个 `*.run`（本地文件或下载链接）并想执行/安装时，按“先获取 → 先检查 → 再备份 → 再确认 → 再执行 → 再留证据”的统一流程走。

## 安全约束（必须遵守）

- `.run` 视为危险操作：未知来源的可执行文件可能改系统配置/覆盖文件/启动服务/导致系统不可用。
- 在给出任何“执行 `.run`”的命令前，必须先提示做一次系统全量备份（转 `istoreos-backup-restore`），并向用户确认：
  - 已备份：回复 `已备份`（或在脚本里用 `CONFIRM_BACKUP_DONE=YES`）
  - 或明确跳过：回复 `跳过备份`（或在脚本里用 `CONFIRM_BACKUP_SKIPPED=YES`）
- 先做只读检查，不要上来就 `chmod +x` 或直接运行。

## 只读检查（推荐一键）

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-run-executable/scripts/inspect.sh" <FILE.run>`

## 获取（当输入是 URL）

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-run-executable/scripts/fetch.sh" -o /tmp/<name>.run <URL>`

## 执行（危险操作，需确认）

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; CONFIRM_RUN_EXEC=YES CONFIRM_BACKUP_DONE=YES sh "$SKILLS_DIR/istoreos-run-executable/scripts/run.sh" <FILE.run|URL> [args...]`
- 或（明确跳过备份）：`SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; CONFIRM_RUN_EXEC=YES CONFIRM_BACKUP_SKIPPED=YES sh "$SKILLS_DIR/istoreos-run-executable/scripts/run.sh" <FILE.run|URL> [args...]`

提示：
- 不要绕过 `scripts/run.sh` 直接 `chmod` 并执行 `.run`。只有在 wrapper 不可用、已经说明风险并获得确认后，才可在设备普通 shell 里手动执行等价命令。
- 建议先尝试 `--help/--version` 获取参数；执行时把输出保存为日志便于回溯。
- 如果你希望 iStoreOS 记录本次安装并支持后续 `is-opkg unrun` 回滚：不带参数执行时优先使用 `is-opkg dotrun`（脚本会自动拷贝到临时文件，避免 dotrun 删除用户原文件）。
