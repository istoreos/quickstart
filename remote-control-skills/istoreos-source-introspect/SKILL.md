---
name: istoreos-source-introspect
description: iStoreOS/OpenWrt 只读证据采集：定向读取 /etc/config、/etc/init.d 与 /usr/lib/lua/luci 相关源码片段（默认脱敏），用于未知问题定位与“以证据为准”的分流。
---

# iStoreOS Source Introspection (Read-only)

当你不确定“服务入口/启用开关/配置字段/LuCI 实际读写逻辑”时，先采集证据再下结论，避免猜。

## 安全约束（必须遵守）

- 只读：不写 `/etc/config`、不改 UCI、不开停服务。
- 默认脱敏：遇到 `password/token/secret/key` 等字段，输出前先脱敏；不要让用户粘贴明文密钥。
- 不要整树 dump `/usr/lib/lua/luci`；只按目标名定向检索。

## 一键采集（推荐）

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-source-introspect/scripts/collect.sh" <name-or-keyword>`

提示：
- `<name-or-keyword>` 通常是 service/uci/meta 名（例如 `dockerd`、`istoreenhance`、`frpc`）。
- 该脚本会尝试收集：UCI 摘要、init 脚本片段、已安装包中的 LuCI 文件清单（若 opkg 可用）、以及 LuCI 目录内的定向匹配结果（小样本）。
