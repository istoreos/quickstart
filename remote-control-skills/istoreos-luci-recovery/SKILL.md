---
name: istoreos-luci-recovery
description: iStoreOS/OpenWrt LuCI Web 界面故障恢复闭环：先只读诊断（uhttpd/rpcd/logread/空间/rom 基线），再软恢复（清缓存+重启服务），必要时重装 LuCI 核心包（需明确确认并建议先全量备份）。
---

# LuCI Recovery (iStoreOS/OpenWrt)

用于处理“安装插件后 LuCI 异常（打不开/500/空白页/登录循环）”。

## 核心原则

- 先证据：先跑只读诊断脚本再下结论。
- 先软恢复：清缓存 + 重启 `rpcd/uhttpd` 往往就能恢复。
- 能不改 `/usr/lib/lua/luci` 源码就不改：优先重装相关包或回退冲突插件。
- 任何写操作（重启服务/清缓存/重装包/改源码）都需要用户明确确认；重装前强烈建议先做系统全量备份（`istoreos-backup-restore`）。

## 只读诊断（推荐）

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-luci-recovery/scripts/diag.sh"`

## 软恢复（需确认）

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; CONFIRM_LUCI_SOFT_RECOVERY=YES sh "$SKILLS_DIR/istoreos-luci-recovery/scripts/soft_recover.sh"`

## 重装核心包（需确认，建议先备份）

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; CONFIRM_LUCI_REINSTALL=YES sh "$SKILLS_DIR/istoreos-luci-recovery/scripts/reinstall_core.sh"`

## 定位坏文件归属包（只读）

当 `logread`/浏览器报错里出现具体 Lua 文件路径时，用它定位属于哪个包，并对照 `/rom` 基线：

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-luci-recovery/scripts/trace_owner.sh" /usr/lib/lua/luci/<path>.lua`

## 生成修复命令候选（只读）

基于“坏文件路径”输出重装/回退的候选命令清单（脚本不执行写操作）：

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-luci-recovery/scripts/suggest_fix.sh" /usr/lib/lua/luci/<path>.lua`
