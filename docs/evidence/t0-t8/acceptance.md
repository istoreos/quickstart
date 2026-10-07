# 局域网设备管理 T0–T8 四机实机验收

> 执行时间：2026-09-26（UTC）  
> 结论：**T0–T7 通过；T8 未通过 24 小时长稳与登录态页面发布门，因此整体为 No-Go。**

## 1. 测试拓扑

| 角色 | 地址 | 实际职责 |
| --- | --- | --- |
| A 主路由 | `192.168.30.1` | DHCPv4 分配方、默认网关、floatip fallback、Quickstart |
| B 旁路由 | `192.168.30.244` | DHCPv4 disabled、floatip main、正常态持有 VIP |
| C 测试客户端 | `192.168.30.93` | 验证旁路由及浮动路线 |
| D 对照客户端 | `192.168.30.7` | 始终以 A 为默认网关 |
| 浮动网关 | `192.168.30.3` | 正常态在 B，B 故障时漂移至 A |

测试前发现 A 的动态池为 `.2–.249`，覆盖 `.3/.7/.93/.244`。为避免测试期间冲突，T2 临时收窄为 `.100–.239`；验收结束后已按快照恢复原配置。原地址池风险仍需由环境管理员另行处理。

## 2. 各里程碑结果

| 里程碑 | 状态 | 主要实机证据 |
| --- | --- | --- |
| T0 基线与恢复点 | 通过 | 四机可达；角色与路由相符；VIP 只在 B；四份 tar 备份可校验 |
| T1 候选部署与角色识别 | 通过（优化 1 次） | A=`local` 可编辑；B=`none_detected` 只读；修复 `dhcpv4=disabled` 被误判为本机 DHCP |
| T2 固定地址与旁路路线 | 通过（优化 1 次） | C 由 A DHCP 获得 `.93`，网关/DNS `.244`；D 保持 `.1`；Desired/Applied/Observed 状态闭环 |
| T3 浮动网关正常态 | 通过 | 三次连续采样均为 A absent、B present；VIP 邻居 MAC 为 B；Quickstart 目标为 `.3/.3` |
| T4 分配浮动路线 | 通过 | 预览只影响 C；续租前为 `pending_renewal`；续租后 C 网关/DNS 均为 `.3` |
| T5 故障转移与恢复 | 通过（优化 1 次） | B LAN 故障后约 31 秒 A 接管；C ARP 自动切到 A；公网/DNS 恢复；B 恢复后 VIP 和 ARP 回归 B；无双主 |
| T6 负向与权限 | 通过（优化 1 次） | stale=`conflict`、中文 DHCP hostname=`validation_failed`、重复 IP=`conflict`、坏目标=`not_found`；B 预览和提交均=`dhcp_authority_unavailable`；配置哈希不变 |
| T7 重启与持久化 | 通过 | A/B 均真实重启；关键配置、二进制和 floatip 脚本哈希不变；服务自启；C 续租保持 `.93/.3/.3`，D 保持 `.1` |
| T8 回归、性能与长稳 | **未通过** | 自动化和即时性能通过；24 小时任务无法在当前执行容器中持久托管，且缺少 LuCI 登录密码，不能完成登录态浏览器验收 |

## 3. T0 恢复点

四台设备均保存：

`/root/quickstart-backups/t0-pretest-20260926T0201Z.tar.gz`

| 设备 | SHA-256 |
| --- | --- |
| A | `a0c4e0f0c84d8fdb1ef6d523e89a0a34f98d47e21b66d72db0bcdfd3357ee01a` |
| B | `5feb0bdfd3c036f6647cc34b09135b68a414aef352b9f43e91a712f4a8936a8c` |
| C | `57fbdeb1eac5585c326848e1e46e83b03b0fbeafb172206bbe6baa7f5d0b447c` |
| D | `875f429c18eef4592cd227356807f69315bbe07b328b2e1b5072bac54739e893` |

阶段快照还包括 A 的 `t2-dhcp.before`、`t2-firewall.before`、C 的 `t2-network-c.before`，以及 A/B 的 T5 floatip 脚本备份。

## 4. 本轮发现并修复的问题

### 4.1 DHCP 分配权判断分叉

- 现象：B 配置 `dhcp.lan.dhcpv4='disabled'`，上下文正确只读，但网关策略预览仍返回 `canApply=true`。
- 原因：上下文使用新判定，旧写路径只检查 `ignore=1`。
- 修复：统一 `ignore`、`dhcpv4=disabled`、`dhcpv4=relay` 的有效 DHCPv4 判定；网关预览与联合写入均在无本机分配权时失败关闭。
- 回归：新增领域辅助函数及网关策略权限测试；B 实机预览与提交均拒绝，且 DHCP 配置哈希不变。
- Quickstart 提交：`70427af fix: enforce DHCP authority on route writes`。

### 4.2 浮动 IP 接管后客户端 ARP 未刷新

- 现象：A 已持有 `.3`，但 C 仍把 `.3` 发往 B 的旧 MAC，导致接管后黑洞。
- 原因：floatip 只执行 `ip addr add`，没有可靠发送 Gratuitous ARP。
- 修复：接管后同时发送 unsolicited ARP request 与 ARP reply；包新增 `iputils-arping` 依赖并升至 `1.1.1`。
- 回归：不清理 C 邻居缓存，接管时 MAC 自动变为 A，恢复时自动变回 B；两方向公网和 DNS 均通过。
- OpenWrt app-hub 提交：`f3bd674 fix: announce floating IP ownership changes`。

### 4.3 部署备份文件名

部署脚本原先会生成字面量 `quickstart.${stamp}.bak`。已修正远端变量展开并加入运维契约，部署后 A/B 均产生独立时间戳备份。

- Quickstart 提交：`c946288 fix: preserve timestamped deployment backups`。

## 5. 自动化与性能

### 5.1 自动化

- 后端完整测试：`1482/1482`。
- 后端 race：`1482/1482`。
- 正式前端契约：`47/47`。
- TypeScript 与生产构建：通过。
- 运维脚本契约、shell 语法、`git diff --check`：通过。
- 前端仍有既存 `::v-deep` 弃用警告，不影响构建或本次功能。

### 5.2 四并发实机 API 压力

每个接口 200 次，共 800 次，请求错误均为 0：

| 接口 | 平均 | 最大 |
| --- | ---: | ---: |
| 设备清单 | 1.1 ms | 16.5 ms |
| 路由器上下文 | 27.9 ms | 43.4 ms |
| 上网路线目标 | 6.0 ms | 10.9 ms |
| 单设备联合策略 | 12.1 ms | 19.4 ms |

Quickstart RSS 从约 `20.5 MiB` 短时升到峰值约 `26.9 MiB`，随后回落到约 `21.8 MiB`；PID 与 `/proc` 启动计数不变。可复用工具为：

- `scripts/ops/lan-device-load-check.sh`
- `scripts/ops/lan-device-topology-soak.sh`

## 6. 正式前端检查

- A 上正式静态资源与本地生产构建完全一致：
  - `index.js`: `ee511cb83044e151039735cc23a8a70a5a60fe16a5f3b7cb3714c7784bd82f4a`
  - `style.css`: `e5601dab878e68e3cc1b3b824c958ffc07273559ddc14dce4f7326706eaff5ee`
  - 图标清单: `393f9be7ca9252a94646a8111814df2a9e07690cd5777392fab85053b36e7309`
- LuCI 资源版本为 `0.14.0-r11`，静态资源 HTTP 200。
- 设备详情、30 图标、中文备注与 DHCP hostname 分离、上网路线、能力降级及实现词隐藏均由 47 项正式前端契约覆盖。
- 新环境的 LuCI 登录并非空密码：空密码 POST 与正式页面均返回 HTTP 403。未擅自修改 root 密码，因此本轮不能补做登录态桌面/移动浏览器截图。

## 7. T8 三次优化与停止原因

1. 更新因新领域错误码而过期的测试断言，并改用仓库真实的 Node 测试入口；完整自动化通过。
2. BusyBox 不支持 GNU `xargs -P`；改为纯 `ash` worker 并发器，800 次实机请求通过。
3. 四机 24 小时采样器功能试跑通过，但普通 `nohup` 子进程被当前执行环境回收；尝试 systemd transient service 时确认容器 PID 1 不是 systemd，无法持久托管。

用户约束为同一里程碑最多优化三次，因此停止，不尝试第 4 种守护方式，也不把单个健康样本伪装成 24 小时通过。

建议在一个不会回收子进程的运维终端执行：

```sh
./scripts/ops/lan-device-topology-soak.sh 86400 300 /tmp/quickstart-t8-topology-soak.tsv
```

完成后应检查：PID/start ticks 不变、API 全为 200、A/B 从未双主、C 始终 `.3/.3`、D 始终 `.1/.1`、两端 ping 均为 `ok`，以及 RSS 不呈持续单调增长。

## 8. 收尾状态与发布结论

- 测试结束后，C 已恢复到初始静态 `.93`、网关 `.244`、DNS `223.5.5.5`；D 保持 `.1`。
- A 的 DHCP 与防火墙已按 T2 前快照恢复；dnsmasq 语法正常。VIP 最终为 A absent、B present。
- Quickstart 修复版和 floatip ARP 修复保留部署在 A/B，服务均运行。
- 原始 DHCP 动态池覆盖固定基础设施地址的风险也随快照恢复而回归，后续应单独调整。

发布判定：**No-Go**。阻塞项是当前候选的 24 小时四机长稳证据和新环境登录态正式页面验收；即时功能、故障转移、负向保护、重启持久化、自动化及短时性能均已通过。
