# M39 四机功能与登录态 UI 验收

> 日期：2026-09-26；状态：通过。M40 的 24 小时长稳、故障转移、可复现发布包和最终五层矩阵不在本文提前宣称。

## 环境与恢复点

- A 主路由/DHCP/Quickstart：`192.168.30.1`；B 旁路由：`192.168.30.244`；C：`192.168.30.93 → B`；D：`192.168.30.7 → A`；浮动地址：`192.168.30.3`。
- A/B 均安装 `luci-app-eqos git-6f9e307-1`；floatip 均为 `1.1.0-1`。
- 四机测试前快照：`/root/quickstart-backups/m39-pretest-20260926T041144Z.tar.gz`。A 的 DHCP、firewall、eqos、floatip 与 Quickstart 策略状态已恢复；四个配置文件逐一与快照 SHA-256 相同。
- LuCI 原密码未知而 SSH 使用密钥。验收时临时设置一次性密码取得登录会话，随即原样恢复 `/etc/shadow`；恢复前后 SHA-256 均为 `f0c616...76ad`，未留下测试密码。

## 登录态正式 UI

- [`visual-check.cjs`](visual-check.cjs) 使用真实 LuCI 会话检查桌面 `1440×900` 和移动端 `390×844`，确认没有回到登录页、没有页面级横向溢出。
- 已覆盖设备列表、D 的“使用管理”详情、分组与计划、局域网设置；截图为 `desktop-devices*.png` 与 `mobile-devices*.png`。
- 四区详情保持“概览 / 设备资料 / 网络与上网 / 使用管理”，流量与诊断仍在概览内；移动端详情抽屉、计划/额度表单和局域网设置可滚动操作。
- 英文 LuCI 环境下，M37 新增的少量文案仍会回退成中文；不阻断功能，但必须在 M40 发布冻结前补齐语言包或明确只发布中文界面。

## 功能证据

### 权限与 DHCP

- A：`lan_gateway_candidate + local`，路线和 DHCP 可编辑。
- B：`downstream_router + none_detected`，路线只读，DHCP `editable=false`；对 Plan 提交返回 `dhcp_authority_unavailable`，零写入。
- A 的现有地址池预检识别 `192.168.30.3` 浮动网关、`192.168.30.244` 网关节点和 `192.168.30.88` 自定义网关冲突，返回 `address_conflict`，未重载 dnsmasq。

### 上网路线、分组、计划和额度

- [`api-check.cjs`](api-check.cjs) 在登录态页面同源执行正式 v2 API：创建 `192.168.30.242` 路线、保持稳定 ID 更新为 `192.168.30.241`、分配给 D、验证无替代项删除被拒绝、以 `default` 替代并删除。
- 临时分组为 C 设置跨午夜时段和月额度；随后增加单设备跨午夜例外和周额度。生效来源为 `system_default → group:m39_group → device`，下一节点为 `2026-09-26T07:00:00Z`，路由器时区为 UTC。
- 删除单设备例外与分组后，groups、devicePolicies、quotas 与导出前基线一致。

### 限速与暂停联网

- 已安装但停用的 eqos 原先无法从设备详情启用；补充 `enable` 的 Plan/Apply 后，实机从 `disabled` 变为 `available`，且未自动保存设备策略。
- D 的 3/5 Mbit/s 策略写入 `eqos.quickstart_speed_84a410b789d2`；`tc` 在 `br-lan` 产生 `class htb 1:11 rate 5Mbit` 和匹配 `192.168.30.7` 的过滤器。
- 首轮暂停只阻断新连接，已建立 conntrack 会话继续通过。第 3 次优化增加 fw4 `chain-prepend forward` 的持久 include；最终实测持续 ping 在保存暂停后立即停止，LAN 到 A 的管理连接仍可用，恢复后 D 公网连通。
- 恢复后 A 的 eqos 为原始 `enabled=0`，临时限速、暂停规则和 include 文件均已移除。

## 三次优化

1. B 被识别为 downstream 时，DHCP 设置不再因 `none_detected` 误开放。
2. eqos 已安装但未启用时，补齐能力启用 Plan/Apply、草稿保留与设备详情入口。
3. 暂停联网规则进入 fw4 建连检查之前，确保既有连接也立即中断；恢复快照同时覆盖 nft include。顺带移除两个修改同一测试全局变量的并行标记，race 自检恢复稳定。

## 自动化与候选一致性

- 后端：`go test ./...` 与 `go test -race ./...` 全部通过。
- 前端：52 项测试、类型检查和生产构建通过。
- A/B 候选一致：后端 `64bebd7e...909d0`，`index.js` `17a3af69...4f9`，`style.css` `aca00ed2...302c`。
- 恢复后 A/B Quickstart PID 单实例；即时 A RSS `29224 KiB`、7 线程、8 FD，B RSS `13056 KiB`、4 线程、8 FD。趋势结论只由 M40 的 24 小时样本给出。

## M40 保留门

- 24 小时四机采样与 RSS 趋势。
- 至少一次 VIP 受控接管和恢复，期间不得双主。
- 包源码、OpenWrt package hash、tarball、实机二进制和静态资源闭合；安装、升级、降级、恢复可复现。
- 补齐英文语言包回退；复核五层矩阵中仍为 partial 的实机边界。没有全部 P0 五层证据时只能给 No-Go。
