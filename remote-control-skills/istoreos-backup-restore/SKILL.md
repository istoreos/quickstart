---
name: istoreos-backup-restore
description: iStoreOS 系统备份/恢复标准流程（系统 overlay 全量备份 + 沙箱 ext_overlay 支持 + iStore 应用列表备份）；用于危险操作前提醒备份，以及用户咨询“如何更安全使用 iStoreOS”时的备份/恢复指引。
---

# iStoreOS Backup / Restore

目标：在危险操作前提供“可执行、可回滚”的备份/恢复流程。

## 使用提示（很重要）

- 如果用户是在设备的普通 shell 里执行命令：优先给出系统绝对路径命令（`/usr/libexec/istore/overlay-backup ...`），不要让用户复制 `sh skills/...` 这种相对路径。
- 在 AI agent 环境里，不要依赖当前目录（`cwd`）找 `skills/`；优先使用外部传入的 `SKILLS_DIR`，再 fallback 到 Codex/OpenCode 常见目录：
  - `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"`

## 两类备份（不要混淆）

1) **系统全量备份（推荐）**：`overlay-backup`
- 备份对象：`/overlay/upper`（若是沙箱环境还可能包含 `ext_overlay/upper`）
- 特点：恢复不需要网络；恢复完成会自动重启

2) **iStore 应用列表/依赖备份（可选）**：`backup`
- 备份对象：iStore 安装的 `app-meta-*` 列表（以及可选的依赖 ipk 重打包）

## 一键检查（只读）

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-backup-restore/scripts/check.sh"`

## 系统全量备份（只写外部存储）

- 推荐（在设备 shell 里也可直接执行）：`/usr/libexec/istore/overlay-backup backup <DIR>`
- 或（在已安装此 skill pack 的 agent 环境里执行）：`SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-backup-restore/scripts/backup_system.sh" <DIR>`
  - `<DIR>` 必须在外部存储上（不能是 `/`、`/tmp`、`/overlay`、`/ext_overlay`）
  - 如果 `<DIR>` 是挂载点根目录（例如 `/mnt/vio3-1`），脚本会默认写入 `<DIR>/istore_backup/`

## 系统恢复（危险操作，会自动重启）

- 推荐使用带确认门槛的 wrapper：`SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; CONFIRM_ISTOREOS_RESTORE=YES sh "$SKILLS_DIR/istoreos-backup-restore/scripts/restore_system.sh" <BACKUP_FILE>`
- 不要绕过 wrapper 直接调用底层 restore 命令，除非已经逐条说明会覆盖系统 overlay 并自动重启，且用户明确确认。

注意：
- 非沙箱环境不会列出/建议恢复沙箱备份文件；要恢复沙箱备份需先进入沙箱环境再执行（以系统实现为准）。
