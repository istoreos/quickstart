---
name: istoreos-service-manager
description: iStoreOS/OpenWrt 服务启停与“插件启用”闭环（/etc/init.d enable/start/status + UCI /etc/config enabled/disabled/enable 字段探测与修复；用于安装后服务不启动、插件未启用等问题）。
---

# iStoreOS Service Manager

目标：把“已安装但没生效/服务没起来/插件没启用”的问题收敛到一个通用闭环：**init.d + UCI enable flag + 验证**。

当操作会“覆盖写配置/删除文件/影响核心服务”时，先提醒用户可做系统全量备份：转 `istoreos-backup-restore`。

## 一键确保（推荐）

- 先说明将执行 `/etc/init.d/<service> enable/restart` 和可能的 UCI enable flag 修改。
- 用户明确确认后才执行：`SKILLS_DIR="${SKILLS_DIR:-${CODEX_HOME:-/config/.codex}/skills}"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="/config/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="$HOME/.agents/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"; [ -d "$SKILLS_DIR" ] || SKILLS_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/opencode/skills"; CONFIRM_SERVICE_APPLY=YES sh "$SKILLS_DIR/istoreos-service-manager/scripts/ensure.sh" <service> [uci_config]`
- 未设置 `CONFIRM_SERVICE_APPLY=YES` 时，脚本必须输出 `need-confirmation:` 并拒绝写操作。

约定：
- `<service>`：`/etc/init.d/<service>` 的名字
- `[uci_config]`：默认等于 `<service>`，也可以指定实际 UCI config 名（对应 `/etc/config/<uci_config>`）

## 工作流（手动版）

1) 确认 init 脚本存在：`ls -la /etc/init.d/<service>`
2) 先做 init.d 级别启用与启动：
   - `/etc/init.d/<service> enable`
   - `/etc/init.d/<service> restart`（或 `start`）
3) 再确认“插件是否启用”（UCI）：
   - `test -f /etc/config/<uci_config> && uci -q show <uci_config> | head -n 120`
   - 常见字段（优先级从高到低）：`enabled=1` / `enable=1` / `disabled=0`
   - 如果找不到启用字段：必须打开 `/etc/init.d/<service>` 查证（不要猜）
4) 验证：
   - `/etc/init.d/<service> status || true`
   - `logread | tail -n 200 | grep -iE '<service>|error|fail' || true`
