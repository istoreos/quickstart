---
name: istoreos-app-search
description: iStoreOS/OpenWrt iStore 应用/插件最小搜索闭环：优先扫描设备本地 /usr/lib/opkg/meta/*.json，兜底使用随 config-dir 分发的 apps.jsonl；输出 Top3 候选（含 why/score/type），用户确认 name 后交给 istoreos-package-manager 安装验证。
---

# iStoreOS App Search (MVP)

用于把“口头描述找插件/应用”变成标准的 Top3 候选 + 用户确认流程。

## 核心约束

- 不做复杂系统：不依赖数据库/向量/RAG；先做“每次查询一次扫”的 MVP。
- 数据源优先设备本地：`/usr/lib/opkg/meta/*.json`（与当前设备可安装集合一致）。
- 兜底：如果不在设备环境或缺少 meta，可使用内置 `apps.jsonl`（构建期预处理产物，随 `config-dir` 分发）。

## 输出合同（必须）

- 输出 Top3 候选：`name/title/type_hint/entry/why/score`
- 多候选必须让用户回复 `name` 确认；确认后转 `istoreos-package-manager` 执行安装与验证闭环

## 一键搜索（推荐）

- `SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; sh "$SKILLS_DIR/istoreos-app-search/scripts/search.sh" "<keyword>" [top]`

说明：
- 该脚本会调用本机 `istore-ai-helper` 提供的 HTTP endpoint：`GET /api/istore/app-search?q=...&top=...`
- 可用环境变量覆盖 server 地址：`ISTORE_AI_HELPER_BASE=http://127.0.0.1:8197`
