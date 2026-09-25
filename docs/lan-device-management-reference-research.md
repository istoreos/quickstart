# 局域网设备管理参考项目研究

> 文档状态：外部项目研究记录；不定义当前产品与领域基线  
> 调研日期：2026-09-24  
> 调研对象：`floatip`、`luci-app-oui`、`luci-app-bandix` 的本地第一方源码和随仓文档  
> 使用原则：提炼产品能力和架构模式，不把参考项目的推测、品牌图标或实现细节直接复制到 Quickstart。

## 1. 范围与结论摘要

本次只依据本地源码，不依据二手文章。三个项目解决的是不同层面的问题：

| 项目 | 真正职责 | 最值得借鉴 | 不应直接照搬 |
| --- | --- | --- | --- |
| `floatip` | 两个节点健康检查后争用一个 LAN 浮动 IPv4 | 将“浮动地址运行状态”与“DHCP 下发网关策略”解耦 | 仅用 ping/curl 的轻量选主、缺少仲裁和完整运行态 API |
| `luci-app-oui` | 离线 MAC 前缀到品牌图标的可选增强 | 最长前缀匹配、随机 MAC 拒识别、精确 MAC 覆盖和失败回退 | 把网卡厂商当成设备品类；直接复用品牌 Logo |
| `luci-app-bandix` | 基于独立 Bandix 后端的流量、连接、DNS、限速和配额 UI | 按设备组织实时流量、历史、规则与配额，桌面表格/移动卡片双布局 | 8,492 行单页、1 秒多路轮询、Shell RPC 大代理和过密功能堆叠 |

综合结论：Quickstart 应继续以“稳定设备身份”为中心，把名称、静态地址、DHCP 标签、网关路径、限速和观测统一成设备策略；浮动网关应是可选择的“网络路径”，而不是暴露给普通用户的底层标签。品牌识别和设备类型必须是两条证据链，流量观测和策略写入也必须是两个边界清楚的后端模块。

## 2. `floatip`

源码根：`/projects/workspace-linkease-ubuntu/openwrt-apps/istoreos-app-hub/apps/floatip`

### 2.1 已确认事实

1. 服务本质是“浮动 IPv4”，只在 LAN 设备上添加/删除地址，不会主动修改 DHCP 通告的网关；LuCI 说明明确提醒了这一点。证据：`luci-app-floatip/luasrc/model/cbi/floatip.lua` 的 `Map("floatip", ...)` 说明；`floatip/files/floatip.sh` 的 `set_up()`、`set_down()`。
2. 配置只有一个 `main` section，核心字段是 `enabled`、`role`、`set_ip`、`check_ip`、`check_url`、`check_url_timeout`。证据：`floatip/files/floatip.config`；`luci-app-floatip/luasrc/model/cbi/floatip.lua`。
3. `role=main` 在中文 UI 中称“旁路由/抢占节点”，`role=fallback` 称“主路由/后备节点”。证据：`luci-app-floatip/po/zh-cn/floatip.po` 的 `FG Preempt Node`、`FG Fallback Node` 翻译。
4. 后备节点通过 `fallback_loop()` ping 一个或多个 `check_ip`；连续三轮找不到存活节点后取得浮动 IP，任一检查地址恢复后释放。证据：`floatip/files/floatip.sh` 的 `fallback_loop()`。
5. `fallback_loop()` 会过滤掉与浮动地址不在同一子网的 `check_ip`，并把最近成功的地址移到检查顺序前面。证据：同文件 `valid_check_ip`、`order_check_ip` 处理。
6. 抢占节点通过 `main_loop()` 检查浮动 IP 是否已被占用，同时用 `curl -I` 检查所有 `check_url`；外部检查成功且浮动 IP 无人占用时取得地址，连续三次 URL 失败后释放。证据：同文件 `main_loop()`。
7. 抢占节点的 URL 健康失败会创建/启用 `firewall.floatip_lan_offline`，丢弃 LAN 到本机的 IPv4 echo-request；健康恢复后禁用该规则。该行为让后备节点把抢占节点视为不可用。证据：同文件 `set_lan_ping()`。
8. 地址切换使用两个 `flock`：一个阻止主循环重复运行，一个序列化地址增删；取得地址后写 `/tmp/run/floatip_cleanup.sh` 供释放。证据：同文件 `try_lock()`、`try_lock_updown()`、`set_up()`。
9. init 脚本只允许静态 LAN，解析 `network.lan.device`，并确认浮动 IP 属于任一 LAN 子网后才启动服务。证据：`floatip/files/floatip.init` 的 `start_service()`。
10. 服务通过 procd 运行并监听 `network`、`floatip` 配置变化；停止时清理浮动 IP 并恢复 LAN ping。证据：同文件 `service_triggers()`、`service_stopped()`。
11. LuCI 状态接口只通过能否非阻塞取得主循环锁判断“进程运行”，每 5 秒轮询；它不表示当前浮动 IP 在哪台节点、健康检查是否成功。证据：`luci-app-floatip/luasrc/controller/floatip.lua::floatip_status()`；`luci-app-floatip/luasrc/view/floatip_status.htm`。
12. 首次安装脚本会删除旧的 `network.floatip` 配置；仅在服务未启用、LAN 为静态且有 LAN gateway 时将角色设为 `main`。证据：`floatip/files/floatip.uci-default`。

### 2.2 可借鉴的产品与架构模式

- 保留“浮动 IP 服务”和“哪些终端拿这个 IP 当网关”的职责分离。前者管理高可用地址，后者由 DHCP 标签/设备策略管理。
- 在 Quickstart 中把组合后的能力命名为“网络路径”或“上网网关”，选项可为“跟随全局、主路由、指定网关、浮动网关”，隐藏裸 DHCP tag。
- 后端应暴露结构化状态：`disabled/starting/healthy/degraded/failover/error`、本机角色、当前持有者、最后切换时间、检查目标及最近结果。
- 配置提交前沿用 `start_service()` 的子网校验，并增加 IP 冲突、两个节点角色互补、检查地址可达性和 DHCP 网关一致性预检。
- 借鉴锁和停止清理思路，但由一个明确的 FloatingGateway 服务模块负责生命周期、健康和事件，不让设备策略模块直接操作接口地址。

### 2.3 限制与风险

- 只有 IPv4；健康模型依赖 ICMP 和 HTTP HEAD，没有仲裁、认证、租约或脑裂防护，不能描述成 VRRP 等价实现。
- 状态页只看进程锁，无法回答“浮动 IP 当前在哪”“为什么切换”；产品整合前必须补齐可观测性。
- `set_lan_ping()` 会持久修改并 reload firewall；异常退出时运行态与 UI 可能不一致，需要启动修复和幂等检查。
- init 未配置 `procd_set_param respawn`；主循环异常退出后不一定自动恢复。
- 配置文件允许多个 `list check_ip/check_url`，旧 CBI 却使用单值 `Value`，不能直接复用其交互。
- `floatip/Makefile` 声明核心包 MIT；`luci-app-floatip/Makefile` 声明 LuCI 包 Apache-2.0。若复制代码，必须按具体文件/包保留版权和许可证；更推荐基于行为契约自行实现。

## 3. `luci-app-oui`

源码根：`/projects/workspace-linkease-ubuntu/github-sources/luci-coolsnowwolf/applications/luci-app-oui`

### 3.1 已确认事实

1. 它不是设备列表、发现服务或分类器，只是 DHCP 主机名旁的本地品牌图标增强；无 daemon、外部查询和定时下载。证据：`README.md` 开头的职责说明。
2. 构建期脚本读取 WH-2099/macdb 的 MA-L、MA-M、MA-S，分别保留 24、28、36 bit 前缀，输出带内容哈希的 `vendors-*.json`。证据：`tools/generate.py::generate()`。
3. 当前 `tools/sources.json` 记录 8,637 个匹配前缀、输入 SHA256、图标来源和实际观察到的厂商别名，可追溯生成物。
4. 运行时 `normalize()` 接受三种完整 MAC 格式，并拒绝非法、组播和 locally administered/randomized MAC。证据：`htdocs/luci-static/resources/oui/oui.js::normalize()`。
5. `lookup()` 按 36、28、24 bit 顺序最长前缀匹配；更具体但未知的分配会在生成阶段写成 `null`，防止错误继承父前缀品牌。证据：`oui.js::lookup()`；`tools/generate.py` 的 “mask a recognized parent” 逻辑。
6. 每页只请求一次本地 JSON，失败也缓存，避免轮询不断重试；品牌 SVG 请求按浏览器缓存共享。证据：`oui.js::load()` 和 `README.md`。
7. 图标先放通用 `computer.svg`，命中品牌后异步替换；品牌图加载失败会退回电脑图，再失败则移除图像。证据：`oui.js::decorate()`、`fallback()`。
8. `/etc/config/oui` 支持按完整 MAC 指定 `vendor` slug，用于 OEM 网卡模块与整机品牌不一致的设备；覆盖不会扩散到相同 OUI 的其他设备。证据：`README.md` 的 “OEM devices” 段；`root/etc/config/oui`；`tools/test_oui.js`。
9. 厂商别名在构建期通过规范化后的锚定正则映射，发生多品牌歧义时生成失败。证据：`tools/generate.py::normalize()`、`vendor_for()`；`tools/vendors.json`。
10. 其 ASUS 规则可把 `ASUSTek COMPUTER INC.` 归一为 ASUS 品牌，但参考实现本身并未据此判断电脑还是路由器。证据：`tools/vendors.json` 的 `asus` patterns；`tools/test_generate.py::test_aliases()`。Quickstart 已另行确认采用“ASUS 厂商默认呈现为网络设备”的产品启发式；这属于我们的可修正规则，不是 `luci-app-oui` 提供的事实。
11. LuCI 集成通过功能探测按需加载脚本与 UCI override，并调用 `decorate()`；未安装包时不发数据库请求。证据：同一源码树的 `modules/luci-mod-status/.../40_dhcp.js::renderHostname()`；`modules/luci-base/root/usr/share/rpcd/ucode/luci` 的 `oui` feature。
12. fnOS 图标来自另一个端口探测启发式：当前 IP/MAC 的探测缓存命中 5666/5667 才覆盖 OUI 图标，不是 OS 身份认证。证据：`40_dhcp.js::isFnosClient()`；`README.md`。

### 3.2 可借鉴的产品与架构模式

- 延续“原始厂商、规范品牌、设备类型”三层模型；OUI 原始事实只提供厂商/品牌证据。Quickstart 可以采用经过产品确认的厂商默认类别（当前 ASUS 默认网络设备），但必须标记为 `manufacturer_default`、可由更强型号证据覆盖，并允许用户修正。
- 品牌规则采用构建期可审查注册表：来源哈希、别名、命中样本、反例和歧义检测都进入测试，运行时保持轻量。
- 采用 36/28/24 bit 最长前缀匹配与 `null` 遮罩，避免把更具体的新分配误判为父分配品牌。
- 对随机 MAC 明确显示“品牌未知/私有地址”，不要伪造品牌；所有异步识别都必须有本地通用形态回退。
- 精确 MAC 覆盖适合作为高级“修正品牌”能力，但应进入统一设备 override 存储，与当前手动设备类型并列并可恢复自动识别。
- 功能探测、一次加载、失败缓存和内容哈希可以用于较大的离线识别数据库。

### 3.3 限制与许可证风险

- OUI 是网卡注册者，不一定是整机品牌；README 自己举例说明 Intel 网卡可能让 PC 显示 Intel。产品必须展示“识别依据”和置信度。
- OEM 生态映射（如模组厂商归入整机生态）可能产生系统性误判，应只用于已验证设备或精确覆盖。
- 当前包主许可证是 GPL-2.0-only；macdb 数据是 MIT；多数 Simple Icons 是 CC0，但 CC0 不授予商标权；补充 Logo 仍明确保留商标权；fnOS 标识归权利人所有。证据：`Makefile`、`LICENSE.macdb`、`LICENSE.md`、`DISCLAIMER.md`、`NOTICE.supplemental`、`NOTICE.fnos`。
- 因此可以借鉴算法和治理流程，但 Quickstart 不应直接复制品牌 Logo 资产；继续使用原创设备形态、文本品牌和可审计别名更稳妥。

## 4. `luci-app-bandix`

源码根：`/projects/workspace-linkease-ubuntu/github-sources/luci-app-bandix`

### 4.1 已确认事实

1. LuCI 包依赖独立 `bandix` 后端、`curl`、`jsonfilter/jshn` 等；前端不是流量采集实现。证据：`luci-app-bandix/Makefile`；`README.zh.md`。
2. 随仓架构文档称后端以 Rust/eBPF 在 TC ingress/egress 采集流量并执行限速，用户态管理设备、实时环和长期存储。该结论来自文档，当前 checkout 不含 Rust 后端源码，不能视为已做源码复核。证据：`ARCHITECTURE.md`。
3. `luci.bandix` rpcd 脚本把 LuCI ubus 调用代理到 `127.0.0.1:8686/api/...`，设置连接/总超时并在连接失败时返回空数据或错误。证据：`luci-app-bandix/root/usr/libexec/rpcd/luci.bandix` 顶部 API 常量及各 handler。
4. 设备接口字段覆盖 MAC、IPv4、IPv6、主机名、接入类型、上联接口、Wi-Fi 信道、最后在线、LAN/WAN 速率与累计值、WAN 限速。证据：同脚本 `get_device_status()` 上方契约注释。
5. 状态页提供在线/总设备数、LAN/WAN/总流量卡片、实时趋势、设备表、设备用量排行和时间增量图。证据：`view/bandix/index.js::render()`、`refreshHistory()`、`updateTrafficStatistics()`、`updateTrafficIncrements()`。
6. 设备列表支持时间范围、简洁/详细模式、字段排序、在线判断、最后在线、接入类型、上联口和 2.4/5/6 GHz badge。证据：`index.js::getTimeRangeForPeriod()`、`sortDevices()`、`isDeviceOnline()`、`buildDeviceUplinkChBadges()`。
7. 桌面端使用表格，移动端生成独立卡片；两者都显示 WAN/LAN 流量、规则/配额与操作。证据：`index.js::updateDeviceData()` 中 `bandix-table` 和 `device-list-cards`。
8. 单设备设置弹窗把自定义主机名与定时限速放在一起；定时规则包含星期、起止时间、上传和下载限制，并支持增删改。证据：`index.js::showRateLimitModal()`、`showAddRuleModal()`；RPC 的 `set/update/delete_schedule_limit()`。
9. 多个同时生效的规则在 UI 中取所有非零限制的最小值，并支持跨午夜判断。证据：`index.js::isRuleActive()`、`mergeActiveRulesLimits()`。
10. 每设备流量配额支持分钟、小时、日、周、月和累计六种窗口，达到配额后标记 WAN blocked。证据：`index.js::createQuotaInput()`、`buildTrafficQuotaCell()`；RPC 的 `get/set/delete_traffic_quota()`。
11. 全局限速采用默认上下行值加“豁免设备白名单”。证据：`index.js::showWhitelistModal()` 相关 UI；RPC 的 `get_rate_limit_whitelist()`、`set_default_rate_limit()` 等。
12. 设备可被删除，并明确提示会删除流量历史；主机名通过 `/api/traffic/bindings` 写入。证据：`index.js` 的 delete confirm 与 `saveHostname()`；RPC 的 `delete_device()`、`set_device_hostname()`。
13. 连接页显示全局 TCP/UDP 和 ESTABLISHED/TIME_WAIT/CLOSE_WAIT，并可按协议、状态分页查看设备流。证据：`view/bandix/connection.js::updateGlobalStats()`、`showFlowsModal()`、`loadFlows()`。
14. DNS 页提供查询/响应、响应时延、Top 类型/域名/设备/DNS server，以及域名、设备、类型、server 的过滤与分页。证据：`view/bandix/dns.js::updateQueries()`、`updateStats()`。
15. 设置页包含监控接口、TC backend/order、历史持久化、实时窗口、邻居刷新、附加子网、连接/DNS 开关、记录上限和数据导出/设备事件 URL。证据：`view/bandix/settings.js::render()`。
16. ACL 将大量查询和变更方法暴露给 `luci-app-bandix` 权限域；菜单按 Status、Connections、DNS、Settings 分页。证据：`root/usr/share/rpcd/acl.d/luci-app-bandix.json`；`root/usr/share/luci/menu.d/luci-app-bandix.json`。

### 4.2 可借鉴的产品与架构模式

- 设备行只保留最有决策价值的实时指标；更完整的趋势、连接、DNS 与历史进入详情抽屉或二级分析页。
- 桌面表格和移动卡片使用同一设备 view model，但组件渲染分开，避免把宽表硬压到手机。
- 把“当前限速”“计划限速”“流量配额”区分为三种不同语义；列表显示结果摘要，完整编辑放到设备详情。
- 借鉴时间规则的星期选择、跨午夜和冲突合并语义，但规则是否生效必须由后端返回，前端只展示，避免客户端时钟导致歧义。
- 借鉴 LAN/WAN 拆分、时间范围、设备排行、最后活跃和数据保留范围；默认页面应渐进披露，不能一次展示全部分析模块。
- 以本机 loopback API 隔离高性能采集进程与 LuCI/Quickstart 对外 API 是可取的，但 Quickstart 应通过类型化后端 adapter，而不是 Shell 拼 JSON。

### 4.3 限制与许可证风险

- 官方 README 明确推荐单网口，复杂 VLAN/企业网络不适用，并要求关闭硬件流量卸载和 Turbo ACC；Linux 6.x、建议 OpenWrt 24.10+。这些前提可能与 iStoreOS 设备能力冲突。
- 硬件交换可能绕过 CPU，因此 LAN 设备间流量不完整；UI 自己也提示这一限制。证据：`README.zh.md`；`index.js` 的 `lanTrafficTooltipText`。
- `index.js` 约 8,492 行、rpcd Shell 约 1,932 行，UI、状态、请求、图表和规则算法高度集中；不应复制这种模块边界。
- 页面同时以 1 秒轮询设备、实时趋势、配额和连接/DNS 数据，容易造成低端路由器 CPU、rpcd 和 DOM 压力。证据：`index.js`、`connection.js`、`dns.js` 中的 `poll.add(..., 1)`。
- Shell 层手工转义和拼 JSON，且不同接口有空对象、透传、`success` 等多种错误形态；Quickstart 应统一错误码、事务和幂等语义。
- 在线安装升级会下载 URL 并调用包管理器，是高权限供应链能力；不属于设备管理核心，不应并入设备详情。
- 删除设备会连同历史删除，必须区分“从列表隐藏”“忘记设备”和“删除统计历史”，避免用户误操作。
- 本 LuCI 仓库根 `LICENSE` 与 `PKG_LICENSE` 为 Apache-2.0；复制实现需保留许可证/NOTICE。独立 Rust `bandix` 后端未包含在本 checkout，其许可证和实现不能由本仓库代推。

## 5. 横向产品结论

### 5.1 应形成的统一设备能力清单

1. 身份与识别：稳定 device ID、名称、MAC/DUID、IPv4/IPv6、连接类型、品牌、设备类型、证据和手动修正。
2. 地址管理：当前地址、历史地址、静态 IPv4，后续可选 IPv6 静态保留；地址冲突和 DHCP 池校验。
3. 网络路径：跟随全局、主路由、指定网关、浮动网关；底层映射为 DHCP tag/gateway，但普通用户不直接编辑 tag。
4. 访问控制：允许上网/暂停上网、按设备策略状态、失败回滚和审计记录。
5. 性能策略：即时限速、计划限速、全局默认加豁免、可选流量配额；统一说明仅作用于可观测/可整形的 WAN 路径。
6. 可观测性：当前上下行、累计量、连接数、最近活跃、短期趋势；高级页再提供排行、长期历史、连接流和 DNS。
7. 高可用网关：配置、预检、节点健康、当前持有者、最近切换、事件时间线和手动测试；不与普通设备操作混在首屏。

### 5.2 推荐的信息架构

- 一级“设备”页：搜索/筛选、在线状态、设备形态、名称、IP、连接类型、当前速率，以及“网络路径/限速/已暂停”等少量状态胶囊。
- 设备详情抽屉：`概览`、`网络设置`、`上网控制`、`流量` 四组渐进披露内容；名称、静态 IP、网络路径在一次保存事务内提交。
- 一级“规则与策略”页：只服务批量审计和复杂规则，不要求普通用户先理解静态列表、限速列表和 DHCP 标签列表。
- 一级“网络设置”页：全局 DHCP/网关/总带宽；浮动网关放在“高级网络路径”，用 A/B 拓扑和实时持有者表达。
- 高级“网络洞察”页：长期趋势、排行、连接和 DNS，按设备详情进入，默认不制造信息噪音。

### 5.3 推荐的架构边界

- `DeviceInventory`：只负责发现、身份合并和在线状态。
- `DeviceClassifier`：消费 OUI/主机名/型号/拓扑证据，输出品牌、品类、置信度和来源；不写策略。
- `DevicePolicy`：以稳定 device ID 管理名称、静态地址、网络路径、访问和性能策略，并做事务/回滚。
- `TrafficTelemetry`：采样、地址归属、实时值和历史聚合；不直接修改设备配置。
- `GatewayProfile`：把 DHCP tag、固定 gateway、浮动 gateway 封装为用户可理解的网络路径。
- `FloatingGateway`：管理节点、健康、地址持有与切换事件，向 `GatewayProfile` 暴露可用性而非 UCI 细节。
- `Audit/Event`：记录策略变更、识别覆盖、地址冲突和网关切换，供 UI 解释“为什么现在是这个状态”。

## 6. 最终取舍

优先吸收的是：`floatip` 的职责解耦、`luci-app-oui` 的离线匹配与可审计回退、Bandix 的设备中心观测维度和响应式呈现。

明确拒绝的是：把 OUI 推断伪装成确定的产品事实、把 DHCP tag 当用户概念、直接打包第三方品牌 Logo、在一个页面堆满监控与策略、用前端时钟决定规则真值、以及把 Shell API 代理继续扩展成业务核心。当前“ASUS 默认网络设备”是已确认、可解释、可人工纠正的产品启发式，不得标成型号级识别。

这份研究应作为后续产品路线图的输入；真正实施前，每一项还应对照 Quickstart 当前代码、测试设备能力和包许可证单独验收。
