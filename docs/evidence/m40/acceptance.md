# M40 长稳、可复现发布与最终门禁

> 日期：2026-09-26；状态：执行中。24 小时采样未到达终点前，本里程碑不得标记完成或给出 Go。

## 冻结候选

- Quickstart 源码提交：`da2f0e9`（`feat: prepare reproducible M40 release`）。
- OpenWrt app-hub 提交：`0c7cfdf`（`feat: package quickstart 0.14.0-r12`）。
- 后端版本：`0.14.0`；LuCI 包版本：`0.14.0-r12`。
- 源码包：`quickstart-binary-0.14.0.tar.gz`，SHA-256 `69b4b011cbd5acf8a604dd32550975940184bbfcf1ec58c6e932d81c4f93e96e`。
- 两次独立三架构构建产生逐字节相同的源码包和二进制：
  - x86_64：`1fea489dbba5077d9cdf72de2c3f0052d3143c555334ebe06bd2a2f119bf9653`
  - aarch64：`0a9f462c7cf7ae217590cdbdc5b611d0e9c2eba9c257e3a2bae676fd1b7761fd`
  - arm：`f67f01284739164582ccd067d678ee178c1e870f52cde5a7faa6fc236aa0a741`

实机 A/B 的 x86_64 后端、Web 静态资源和 `main.htm` 已与上述候选一致。正式采样开始后冻结代码和服务进程。

## UI 与语言边界

- 登录态英文界面覆盖设备、使用管理、分组与计划、局域网设置；桌面 `1440×900` 和手机 `390×844` 均无页面级横向溢出。
- 缺失或 `fuzzy` 的设备管理英文词条会直接令正式 UI 测试失败；内置路线名和运行时策略摘要使用显式翻译，用户自定义名称保持原文。
- 截图和可重复检查脚本位于本目录的 `desktop-*.png`、`mobile-*.png` 与 `visual-check.cjs`。

## 安装、升级、降级和恢复

- A/B 升级前备份分别为 `/root/quickstart-backups/m40-pre-release-20260926T050000Z.tar.gz`。
- 候选包解压和独立 `quickstart version` 通过；A/B 升级后单实例且 API 返回 200。
- A/B 降级到 M39 后端与 Web 备份后 API 返回 200；重新恢复 M40 候选后 API 返回 200。
- 设备分组、策略效果、任务事务和审计配置在整个生命周期中哈希不变。`traffic-insights.json` 是运行时遥测，会随采样自然变化，不作为配置丢失判据。

## 四机 SMOKE 与有界负载

- `scripts/ops/lan-device-topology-smoke.sh` 是只读、失败关闭的一键四机检查；覆盖 A/B SSH 与单进程、后端和静态资源哈希、页面资产版本、软件包版本、Quickstart/dnsmasq/floatip、8 组 GET API、路由角色与 DHCP 权限、VIP 单主，以及 C/D 默认路线、公网、VIP 和 ARP 所有者。
- 候选运行时模式 `REQUIRE_PACKAGE_COHERENCE=0`：48/48 通过。A 为 `lan_gateway_candidate + local` 且可编辑；B 为 `downstream_router + none_detected` 且失败关闭；VIP 仅由 B 持有；C 经 B、D 经 A，二者公网和 VIP 可达。
- 正式软件包模式：52 项中 4 项失败。A 的 `quickstart/luci-app-quickstart` 仍登记为 `0.11.6-r1`，B 分别为 `0.9.9-r1`、`0.8.17-r1`，而验收要求是 `0.14.0-r12`。因此只能确认 M40 候选文件已部署，不能确认 r12 软件包安装完成。
- A/B 各执行 4 个只读接口、每接口 100 次、4 worker 的有界并发，共 800 次请求，HTTP 错误为 0；A/B PID 与启动时钟前后不变。单次最慢请求 A 为 `0.0419s`、B 为 `0.0240s`。负载后的 RSS 回落和长期趋势继续由 24 小时门禁判定。

## 浮动网关受控切换

- 初始由 B 持有 `192.168.30.3`。停止 B 的 floatip 并仅阻断其探测响应后，A 在 3 次检查后接管；恢复 B 后 A 释放、B 重新持有。
- 共 120 个约秒级样本：双主样本 0；首次接管和恢复期间无主样本 12；A 持有 33、B 持有 75。
- 切换期间 C/D 均能访问浮动地址；C 的默认网关保持 `192.168.30.244`，D 保持 `192.168.30.1`。恢复后 B 的 firewall 配置与演练前逐字节一致。

## 24 小时稳定性

- 原始采样：`/tmp/quickstart-m40-topology-soak-final.tsv`，间隔 300 秒，起点 `2026-09-26T05:28:26Z`。采样器运行在持久 tmux 会话中，不依赖当前交互连接。
- 每个样本记录 A/B PID、启动时钟、RSS、线程、FD、API 状态与耗时、VIP 持有者，以及 C/D 默认网关、DNS 和公网连通性。
- `scripts/analyze-lan-device-soak.mjs` 失败关闭地检查真实时长、进程重启、API 错误、双主/无主、路线漂移和持续 RSS 增长。
- 当前结论：**待 24 小时结束后填写**。

## 发布门禁与已知阻塞

- OpenWrt package 的 `PKG_HASH`、版本、静态资源和本地源码包已经闭合，package contract 测试通过。
- package 当前配置的上游下载地址返回 HTTP 404；在发布 tarball 到该既有外部 release 前，不能宣称 OpenWrt buildroot 可从远端完成构建。
- 五层矩阵只把 M39 已有直接实机证据的条目改为 `verified`；其余未覆盖边界保持 `partial`，不以相邻场景推断通过。
- 最终规则：24 小时门禁和全部 P0 五层证据均为绿色才是 Go，否则结论为 No-Go 并列出下一批可执行缺口。
