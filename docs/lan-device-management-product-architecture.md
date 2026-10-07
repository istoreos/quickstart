# 局域网设备管理：产品与领域基线

> 状态：**当前唯一产品与领域基线（M64 业务验收候选）**
> 基线版本：M64 / 2026-09-28
> 范围：Quickstart `backend` / `web`、OpenWrt `quickstart` 集成、`floatip`、`luci-app-oui`、`luci-app-bandix`  
> 文档权限：本文件定义产品语义、领域语言和模块 Interface；研究、原型、历史里程碑与能力对比不能覆盖本文  
> 关联资料：[能力对比与历史路线图](device-management-comparison-and-roadmap.md)、[低保真交互](lan-device-management-low-fidelity-design.md)、[设备品牌注册表](device-brand-registry.md)

> M11～M20 的历史任务与证据见 [《局域网设备管理 M11～M21 实施里程碑》](lan-device-management-m11-m21-milestones.md)；M21～M31 见 [《基线冻结、P0 闭环与正式 UI》](lan-device-management-m21-m31-baseline-p0-ui-milestones.md)；当前收口顺序见 [《M32～M40 收口里程碑》](lan-device-management-m32-m40-closeout-milestones.md)。

## 1. 结论

Quickstart 的核心产品是“管理局域网中的设备及其上网方式”，不是若干 DHCP、限速和防火墙配置表。正式产品只采用“设备 / 分组与计划 / 局域网设置”三个一级区域；规则台账属于局域网设置中的批量与离线维护工具，不是第四个一级产品。

M11～M20 已实现的能力继续作为迁移输入，但经典静态/限速页面、裸 DHCP 标签和旧接口不是目标产品的一部分。它们只允许被一次性迁移器读取，不建立长期双写或双 UI。

所有功能必须落入三个用户需求层级：

1. **看见并理解设备**：这是谁、是否在线、从哪里接入、现在用了多少网络；
2. **管理设备如何联网**：地址、上网路线、联网权限、速度、时段和额度；
3. **配置提供这些结果的局域网能力**：DHCP、路线目标、浮动网关、总带宽和可选增强能力。

底层继续使用 DHCP host/tag、dnsmasq option、eqos、防火墙和 floatip，但普通用户不再需要理解这些实现名词。

### 1.1 七个 P0 的冻结结论

| P0 | 冻结结论 | 可观察结果 |
| --- | --- | --- |
| P0-1 信息架构冲突 | 唯一一级结构是“设备 / 分组与计划 / 局域网设置” | 不存在第二套目标导航 |
| P0-2 长期兼容冲突 | 只做一次性、幂等、可预览和可恢复的迁移 | 无经典 UI、双写或长期旧 Interface |
| P0-3 权限依赖错误 | Internet Access Policy 与 Speed Policy 独立 | eqos 缺失不影响暂停/恢复联网 |
| P0-4 事务范围过大 | 使用设备资料、网络与上网、使用限制三个任务级事务 | 任务内恢复；跨任务不承诺伪原子性 |
| P0-5 路线效果失真 | Desired Policy、Applied Configuration、Observed Effect 分开 | 保存成功不显示为终端已生效 |
| P0-6 DHCP 权限不明 | Router Context 按选定 LAN 给出 DHCP authority | 非本机分配时路线只读引导 |
| P0-7 图标模型缺失 | Device Profile 持久化品牌/类别修正和 Icon Preference | 手动图标不被自动识别覆盖，可恢复自动 |

### 1.2 三个用户写入任务

| 写入任务 | 包含字段 | 事务范围 | 失败语义 |
| --- | --- | --- | --- |
| 设备资料 | Unicode 备注、品牌修正、类别修正、自动/手动图标偏好 | 仅 Device Profile | 原子保存或保持旧资料；不触碰 DHCP、分组或网络策略 |
| 网络与上网 | 地址模式、地址预留、ASCII DHCP Hostname、上网路线；DNS 固定跟随路线 | DHCP host、地址及匹配的网关/DNS配置 | 整个任务恢复到旧网络配置，或明确进入“需要恢复”；不产生半条路线 |
| 使用限制 | 联网权限、限速、时段、额度和单设备例外 | firewall、eqos、计划与额度的补偿事务 | 写入前检查所有被修改能力；完全恢复或明确逐项报告“需要恢复” |

三个任务分别保存。页面可以统一展示，但不得用一个“保存全部”制造跨 dnsmasq、firewall、eqos 和资料文件绝对原子的假象。

### 1.3 M32～M38 增量冻结结论

- “上网路线”是唯一用户领域对象；创建、编辑、删除和引用迁移均使用稳定 Target ID 与 Plan/Apply，普通界面不得重新出现 DHCP tag、option 3/6 或 UCI。
- DHCP 开关、地址池、租期和默认路线属于同一局域网服务任务。路由器地址、网络/广播地址、浮动 IP、网关节点和静态预留冲突必须在写入前阻止；无本机 DHCP 分配权时只读。
- 设备详情只有“概览 / 设备资料 / 网络与上网 / 使用管理”四个一级区域。历史流量和诊断进入概览的按需内容；时间计划与额度进入使用管理。
- Capability 只有 `available / disabled / not_installed / unsupported / error` 五态；未知值按 `error` 失败关闭，不能推测为可用。
- Webhook 只外发事件、时间和稳定伪匿名设备标识，不外发原始 MAC、DNS 内容或本地审计原因。管理入口探测与策略包导入是独立模块，不属于能力状态聚合器的写职责。
- 当前 30 个原创通用图标表达设备形态，品牌继续用文字表达；不嵌入第三方商标 Logo。

## 2. 当前完整功能清单

下表是防止后续重构丢功能的产品台账。“当前入口”描述用户现在在哪里使用；“保留要求”是今后改版不可回归的能力。

| 领域 | 当前能力 | 当前入口 / 实现 | 当前不足 | 保留要求 |
| --- | --- | --- | --- | --- |
| 设备发现 | 合并 DHCPv4、ARP、NDP、DHCPv6、host hints、Wi-Fi 关联与本次开机历史 | 新设备中心；`DeviceInventory.Snapshot` | 历史不跨重启，来源诊断对普通用户仍偏隐蔽 | 单一设备身份、多地址和部分数据源失败降级不能退化 |
| 在线状态 | 在线、离线、最近出现、当前/历史地址 | 新设备中心列表与详情 | 缺少“首次发现”、离线时长筛选和永久历史 | 在线不能与“允许联网”混为一谈 |
| 设备识别 | 原始厂商、短品牌、设备类别、来源、可信度、人工修正；ASUS 厂商默认网络设备 | 列表身份区、详情分类编辑器；`DeviceClassifier` | 规则包仍为内置版本，尚无签名在线更新 | 品牌与类别可解释、可修正；未知设备必须安全回退，不参与身份合并 |
| 场景图标 | 30 个原创通用设备形态，缺图回退 | `DeviceSceneIcon`、`DeviceProfileEditor` | 品牌只靠文字；类别误判时仍需人工修正 | 不复制第三方 logo；品牌文字始终可见，图标只表达形态 |
| 查找与浏览 | 在线/全部/已控制筛选，名称/IP/MAC/厂商搜索，连接方式/品牌/策略组合筛选，最近活跃/名称排序，桌面表格与移动卡片 | 新设备中心 | 当前不提供批量选择，避免把低频操作堆入首页 | 默认信息密度保持简洁；高级筛选渐进展开；筛选不能丢失详情上下文；移动端不得退回横向大表 |
| 实时与历史流量 | 每设备速率、运行期累计、连接数、短时趋势；小时/日/月聚合和日/周/月配额 | 设备详情；`DeviceTraffic` + `TrafficInsights` | 详细连接/DNS 需要经验证的可选 Adapter；当前不因检测到 Bandix 可执行文件就宣称可用 | 未知数据不得伪装成 0；无配额时无后台采样，启用配额时仅每分钟采样；历史固定预算和节流写入 |
| 静态地址 | 添加、编辑、删除、批量删除；MAC、IPv4、可选 MAC-IP 绑定 | 详情短表单、静态分配列表；DHCP UCI | 设备名称与静态租约耦合；只支持 IPv4 写入；表单允许直接写 DHCP 标签 | 离线规则维护和批量删除必须保留；冲突检查与回滚不能退化 |
| 设备名称 | 独立 Unicode 设备备注名；DHCP 主机名为可选严格 ASCII 单标签 | 设备详情、静态地址高级项 | boot-local 身份备注仅本次开机有效 | 两类名称始终分离；DHCP 写入必须后端校验、typed UCI、预检和回滚 |
| 设备上网路线 | 通过 Gateway Target 选择默认/本机/上级/旁路由/浮动网关；内部编译为 option 3/6 | 设备详情“地址与上网路线” | DHCP 生效仍取决于终端续租 | 不暴露 tag；明确 pending renewal/active/unverifiable/error |
| DHCP 全局设置 | DHCP 开关、地址池、租期和默认路线的安全预览与应用 | 局域网设置 → 网络服务 | 实机仍需覆盖关闭、冲突恢复和外部 DHCP 组合 | 危险确认、受影响设备数、冲突验证与回滚必须保留 |
| 上网路线目标 | 新建、编辑、删除旁路由/指定 IP 路线；统计设备、分组、全局与 LAN 默认引用 | 局域网设置 → 上网路线 | M39 已完成创建、稳定 ID 更新、引用替代和删除的登录态实机复验 | 使用中删除必须选择替代路线；内部 tag/option 不进入产品接口 |
| 浮动网关 | 启用、主/备用节点角色、虚拟 IP、对端 IP、URL 探测参数；配置改变时清理自动 DHCP tag/host | 全局设置 → 浮动网关；`floatip` + Quickstart Adapter | 角色名称容易反直觉；缺少实时持有者、健康、最近切换、双端配置一致性 | 浮动 IP 漂移能力与 DHCP 广播职责必须分开表达；配置必须可诊断、可演练 |
| 限速服务 | 原生引擎默认可用；MAC 身份统一覆盖 IPv4/IPv6；兼容模式可配置总上传/下载带宽；既有 Bandix 可显式迁移 | 局域网设置 → 网络服务 → 高级执行方式 | 原生实现是 TC policing 上限控制，不承诺队列整形的精确恒速 | 技术 Provider 默认折叠；不静默切换、不与 Bandix 同时接管接口；界面表达“最高速度” |
| 单设备限速 | 按 MAC 设置上传/下载上限，支持添加、修改、删除和规则台账 | 详情 → 使用管理、局域网设置 → 规则台账 | 当前只支持固定上限；时段与额度由独立策略层组合 | 详情快捷控制与离线规则台账均保留；保存、加载、验证状态不得混同 |
| 联网权限 | 按设备阻断/恢复网络，危险确认显示设备身份 | 详情 → 使用管理；防火墙 | M39 已证明立即阻断和恢复且不依赖 eqos；完整能力异常组合仍以五层矩阵为准 | 在线状态与联网权限分开；恢复后不应误删仍需生效的限速 |
| 规则台账 | 地址、路线、限速、断网和孤立规则统一筛选/批量预览 | 局域网设置 → 规则台账 | 路线生命周期已闭环；批量重载和其他修复动作仍需补充实机证据 | 离线规则、冲突定位、批量维护和逐字段删除语义不能退化 |
| 能力与依赖 | 统一五态并支持安装 floatip/eqos；未知状态失败关闭 | 全局设置和详情策略 | eqos 可用态仍待匹配固件实机验证 | 依赖不可用不能拖垮设备清单；所有入口显示同一能力状态 |
| 写入安全 | 策略、路线目标、DHCP、规则、分组和导入均有校验、版本、快照、验证和失败回滚 | 各领域深模块与 typed store | M40 已完成发布候选升级、降级和恢复；服务失败及回滚失败注入仍需补充实机证据 | 高影响变更必须 plan → validate → apply → verify → rollback |
| 分组与计划 | 独立设备分组、优先级、跨午夜时段、全局→组→设备来源解释 | 设备分组面板；`DeviceGroups` | 当前为分钟级单循环，不做复杂日历 | 分组不得复用 DHCP tag；循环、成员、策略、事件均保持上限 |
| 审计与备份 | 新设备/上下线/策略/配额事件、有界伪匿名 Webhook、策略导出与 dry-run 导入 | 概览 → 诊断与高级工具；`NetworkAudit`、`ManagementProbe`、`PolicyBundle` | 备份范围当前只含分组、设备例外与配额 | 原始 MAC/DNS/reason 不外发；导入必须明确范围、确认和回滚 |
| 迁移输入 | 唯一正式 UI 已启用；旧配置只由一次性迁移器读取 | 迁移卡片与旧配置 | M39 尚需补齐首次执行、冲突和恢复实机证据 | 不恢复旧入口、旧写入或双写 |
| OpenWrt 集成 | 后端 `0.14.0` 与 LuCI 候选 `0.14.0-r12` 已实机对齐，包含 30 个 WebP 场景图标、`device_inventory_v2` 开关和分类覆盖 conffile | `apps/quickstart` 的两个 Makefile、`main.htm`、安装树 | 本地可复现 tarball 与 `PKG_HASH` 已闭合；既有远端下载地址仍为 HTTP 404，需发布资产后才能完成 buildroot 远端构建 | 页面源码、构建产物、包版本、配置保留和回滚契约必须一起验收 |

## 3. 当前真实行为链

### 3.1 发现、展示与控制

```text
DHCP / ARP / NDP / Wi-Fi / host hints
                 │
                 ▼
        DeviceInventory.Snapshot
                 │ stable deviceId + current owner
       ┌─────────┴──────────┐
       ▼                    ▼
DeviceClassifier       DeviceTraffic
       └─────────┬──────────┘
                 ▼
            设备中心列表
                 │
                 ▼
          DevicePolicy.Get/Apply
        ┌────────┼───────────┐
        ▼        ▼           ▼
     DHCP UCI   eqos       firewall
```

这个分层是现有架构的优点：设备身份、时效遥测和期望策略已经是不同概念。下一步应扩充统一策略，而不是再从前端直接调用更多 UCI 细节。

### 3.2 设备选择浮动网关的真实链路

```text
两台 Gateway Node 运行 floatip
          │ 健康检查与 IP 漂移
          ▼
     Floating Gateway（虚拟 IPv4）
          │
Quickstart 创建 DHCP Policy Tag（option 3 + option 6）
          │
设备的 DHCP host 规则引用该 tag
          │ 重新获取租约后
          ▼
设备获得“虚拟 IPv4 作为网关和 DNS”
```

必须保留两个事实：

- `floatip` 只让虚拟 IP 在两个路由节点间漂移，不会自动修改 DHCP；
- 给设备选择网关是另一项 DHCP 策略，通常要等设备续租或重新联网才生效。

因此，产品不能只显示“保存成功”，而应区分“浮动网关已配置”“当前由谁持有”“设备策略已保存”“等待设备续租生效”。

## 4. 三个参考项目带来的结论

### 4.1 `luci-app-oui`：学习本地、确定、可回退的识别链

它提供的是本地 OUI 数据和图标装饰，不是完整设备管理：支持 24/28/36 位最长前缀匹配，忽略随机/组播/非法 MAC，数据库和图标按页面/品牌缓存，未知厂商回退通用电脑，还允许按完整 MAC 覆盖 OEM 模块误判。

值得吸收：

- OUI 查询必须完全本地化，不能上传家庭设备 MAC；
- 厂商、消费品牌、设备类型是三件事；
- 用完整 MAC 做人工/OEM 修正，不能污染整个厂商前缀；
- 失败时保留名称和通用图标，而不是阻塞列表。

不能直接复制：其 JavaScript 是 GPL-2.0-only；品牌 SVG 还涉及各自素材声明与商标边界。Quickstart 应延续原创类别图标 + 品牌文字，或通过独立可选包形成许可证清楚的 Adapter。

### 4.2 `luci-app-bandix`：学习时间尺度和设备洞察

Bandix 的价值不只是“实时网速”。它把数据分成实时、天、周、月等时间尺度，并提供 IPv4/IPv6 流量、连接统计、DNS 查询、持久化历史、设备命名、速率限制和按分钟/小时/天/周/月/累计的 WAN 配额。其后端以 Rust/eBPF 实现，项目文档明确主要适合单网口、简单家庭网络，并要求较新的内核且关闭部分卸载能力。

值得吸收：

- 列表只显示当前状态，历史趋势进入详情；
- “速度上限”和“流量配额”是两个不同的用户目标；
- 连接数、DNS、历史流量属于诊断/洞察，不应挤进默认设备行；
- 高性能采样后端应通过 Adapter 接入，设备中心不绑定某一种采样技术。

不应盲目照搬：eBPF、内核版本、单网口和硬件卸载限制会显著缩小兼容范围。应先定义 `TrafficHistory` / `Quota` Interface，再决定使用现有 conntrack、Bandix 或其他实现。

### 4.3 `floatip`：学习职责分离和状态可观测

`floatip` 的 `main` 是优先持有虚拟 IP、并通过 URL 判断自身上网能力的节点；`fallback` 在对端 IP 连续不可达时接管。Quickstart 当前为适配用户拓扑，将 `fallback` 显示为“主路由”、`main` 显示为“旁路由”，这虽然可以工作，却让代码值、网络角色和产品名称方向相反。

下一版不应让用户选择 `main/fallback`，而应询问：

- 本机是“优先提供上网的节点”还是“故障时接管的节点”；
- 虚拟网关 IP 是什么；
- 对端节点地址是什么；
- 优先节点用哪个目标判断互联网正常。

运行态还需显示当前持有者、对端可达性、互联网探测结果、最近切换时间和失败原因。

### 4.4 横向能力对比

`luci-app-oui` 和 `floatip` 都是单一能力组件，因此表中的“无”只表示该组件不负责，并不表示整个 LuCI/OpenWrt 生态没有对应功能。

| 能力 | Quickstart 当前 | `luci-app-oui` | `luci-app-bandix` | `floatip` | 我们的决策 |
| --- | --- | --- | --- | --- | --- |
| 设备聚合与在线状态 | 有，IPv4/IPv6、多地址、本次开机历史 | 无，只装饰已有主机名 | 有，统一设备管理与在线/离线 | 无 | 保持 Quickstart Inventory 为唯一身份来源 |
| 本地 OUI 与厂商 | 有厂商文本、品牌规范化、可解释分类 | 强项：最长前缀、品牌 SVG、完整 MAC 覆盖 | 非核心 | 无 | 借鉴本地匹配与覆盖；不复制 GPL 实现或品牌素材 |
| 设备类型与原创场景图标 | 有 30 个可选形态、可人工修正 | 无类型推断，未知回退电脑图标 | 主要呈现设备信息 | 无 | 保持类别与品牌分离；修正已确认的 ASUS 默认语义 |
| 实时速率与累计 | 有实时、运行期累计、连接数 | 无 | 强项：eBPF、IPv4/IPv6、实时与持久统计 | 无 | 核心页面继续轻量实时；历史通过可选 Adapter |
| 天/周/月历史 | 有，轻量本地分层存储，固定 2 MiB 预算 | 无 | 有，粒度和诊断更丰富 | 无 | 保持本地实现为兼容基线，Bandix 作为可选增强 |
| DNS 与连接洞察 | 仅连接数 | 无 | 有详细连接、DNS 查询和统计 | 无 | 只放详情/诊断，不进入默认列表 |
| 静态地址 | 有 IPv4 规则、批量维护 | 无 | 可读取 DHCP/DNS 名称，但非静态租约管理器 | 无 | 保留并与 Device Alias 解耦，后续补 IPv6 |
| 单设备限速/断网 | 有，限速默认由独立 quickstart-netpolicy 执行，断网使用防火墙 | 无 | 有 eBPF 速率限制 | 无 | Usage Policy 统一解释；Internet Access 与 Speed Capability 独立降级；Bandix 只作显式迁移输入 |
| 流量配额 | 有，日/周/月 notify/block，周期恢复先前联网状态 | 无 | 强项：多时间周期配额、超额阻断 | 无 | 已以稳定身份实现；继续验证计数精度和闪存预算 |
| DHCP 网关分配 | 有全局与设备 tag | 无 | 无 | 明确不负责 | 建立 Gateway Profile，隐藏 tag/option |
| 网关高可用 | 有配置集成，但运行态不足 | 无 | 无 | 强项：虚拟 IP 漂移 | 增加状态、预检、双端向导和演练，不重写核心漂移算法 |
| 设备备注名 | 有独立 Unicode Device Alias，与 DHCP Hostname 解耦 | 支持完整 MAC 的品牌覆盖，不是备注名 | 有 hostname binding | 无 | 保持 Profile 独立，不参与 DHCP、身份合并或流量归属 |
| 可选依赖降级 | 有 capability 状态 | 包不存在时不加载 | 服务依赖，兼容条件较强 | 独立插件 | 任何插件失败都不能影响设备清单 |

### 4.5 Quickstart 的优势与主要短板

我们的优势：

- **闭环最完整**：同一产品里同时具备识别、实时状态、静态地址、网关选择、限速和断网；
- **家庭主/旁路由场景独特**：DHCP 网关策略与浮动网关能组合成设备级高可用上网路线；
- **身份基础更可靠**：新模型已区分稳定身份、当前/历史地址、遥测和策略；
- **兼容性更宽**：核心设备清单不要求 eBPF、Linux 6.x 或单网口；增强组件缺失时可降级；
- **产品形态已优于传统 LuCI 表格**：响应式设备卡片、详情抽屉、明确的加载/错误/预热状态和危险操作确认已经存在。

M38 后仍需处理的主要短板：

- **IPv6 策略仍不等价**：可以识别、展示和统计 IPv6，但目标固件的 IPv6 静态预留写入契约尚未验证，因此界面诚实显示 `unsupported`；
- **高级诊断依赖可选组件**：详细连接和 DNS 洞察需要 Bandix，当前本地基线只提供连接数和受限历史；
- **高可用已通过受控四机演练但长期观察独立延期**：T0～T7、M40 与 M64 即时门均已证明双节点接管、ARP 更新和恢复，受控演练无双主；24 小时无双主与资源趋势必须在最终候选冻结后单独执行；
- **业务验收候选已完成，正式发布尚未完成**：唯一正式 UI、登录态验收、可复现 tarball/`PKG_HASH`、升级/降级/恢复和 55 项五层证据已完成；正式发布仍需最终候选的 24 小时观察，并将源码包发布到既有远端地址；
- **品牌能力保持克制**：已有 ASUS 等品牌文字和原创类别图标，但没有引入第三方商标 Logo 或在线 MAC 查询；
- **网络隔离与多 WAN 是独立能力**：没有合适 Adapter 时只报告 `unsupported`，不把 DHCP 网关分配包装成这两种功能。

## 5. 目标产品信息架构

### 5.1 一级结构

设备管理页面只保留三个一级区域：

| 区域 | 默认用户任务 | 内容 |
| --- | --- | --- |
| 设备 | 找到一台设备并管理它 | 在线/全部/需关注筛选、搜索、设备详情、快捷控制 |
| 分组与计划 | 让多台设备按相同规则运行 | 设备分组、共同路线、联网时段、限速与额度、批量预览和来源解释 |
| 局域网设置 | 配置网络基础能力和处理离线规则 | 网络概况、DHCP、上网路线目标、浮动网关、带宽与统计、规则台账、诊断与高级 |

“静态分配列表”和“限速设备列表”不再是一级产品概念，其离线维护、批量删除和冲突定位能力迁移进“局域网设置 → 规则台账”。规则台账按用户结果展示地址、路线、联网权限、限速和额度，不暴露底层配置对象。

### 5.2 设备列表

默认桌面仍保持六组信息：设备、状态、地址、连接、实时流量、规则摘要。新增能力通过筛选和详情承载，不加新列。

建议筛选：

- 在线、全部、离线；
- 已设置规则、需要关注；
- 有线、Wi-Fi；
- 设备分组（实现后）。

“需要关注”只包含有明确行动的状态，例如地址冲突、网关目标不可用、策略等待续租、流量采样异常。它不能成为泛化的告警垃圾桶。

### 5.3 设备详情

右侧抽屉先给摘要，再按四个用户任务分区：

1. **概览**：当前速率、运行期累计、历史用量和趋势；诊断与高级工具默认折叠；
2. **设备资料**：Device Alias、品牌和类别修正、图标偏好；原始厂商和识别依据只在说明中出现；
3. **网络与上网**：当前/历史地址、地址预留、DHCP Hostname 高级项、上网路线与路线状态；
4. **使用管理**：联网权限、速度上限、时间计划、额度、当前结果、规则来源与下一计划节点。

最常用的“改名、固定地址、选上网路线、限速、断网”均应在打开详情后的三次主要交互内完成。

### 5.4 名称与标签重新定义

当前最容易继续制造产品债务的是两个词：

- **名称**拆成“设备备注名”和“DHCP 主机名”。设备备注名允许安全 Unicode、中文、空格和常用符号并独立保存；DHCP 主机名放在高级设置，只允许严格 ASCII host label。中文不得直接写入 DHCP Hostname，也不自动转拼音或 Punycode；
- **标签**拆成“设备分组”和“DHCP 策略标签”。用户在普通界面只看到设备分组与上网路线，内部标签仅在高级诊断显示。

旧字段只允许迁移器读取；正式 Interface 和界面不得继续写入裸 `tagName`。

### 5.5 上网路线交互

设备详情使用单选卡或下拉框：

```text
上网路线
● 跟随网络默认（当前：本机路由）
○ 本机路由
○ 上级路由  192.168.9.1
○ 旁路由    192.168.9.2 · 正常
○ 浮动网关  192.168.9.3 · 当前由旁路由提供
```

保存前显示是否需要设备重新获取地址；DNS 默认跟随所选路线，不增加第二个普通用户开关。保存后分别展示“期望设置”“服务器配置”和“终端观察”，没有终端代理时最多确认新租约已经出现，不得把它写成“路线已生效”。Gateway Target 删除前显示引用设备数量；有引用时先引导迁移，禁止无提示删除。

### 5.6 浮动网关设置向导

使用三步向导代替一个技术表单：

1. **节点职责**：优先服务节点 / 故障接管节点，并用拓扑小图说明；
2. **地址与检测**：虚拟网关、对端地址、互联网检测目标；自动校验同网段、IP 占用、URL 和双端角色；
3. **预检与启用**：显示将修改的服务、已有引用和回滚点，再执行并验证。

配置完成后的首页是状态卡，不是表单：`健康 / 已接管 / 降级 / 配置不一致`，并显示当前持有者和最近切换。

## 6. 必要补充能力

### 6.1 必须优先补齐

| 能力 | 为什么必要 | 最小可用范围 |
| --- | --- | --- |
| Device Alias | 用户命名不能依赖静态租约，也不应禁止中文 | 以持久 Device Identity 保存、原子写入、可清除 |
| Gateway Profile | 隐藏 DHCP tag/option，统一默认、旁路由、浮动网关与自定义网关 | 列表、健康、引用计数、分配、迁移、删除保护 |
| 浮动网关运行状态 | 只有配置没有状态，用户无法相信故障切换 | 当前持有者、对端/互联网健康、最近切换、原因 |
| 策略生效状态 | DHCP 规则保存不等于终端已经续租 | desired / applying / active / unverifiable / error |
| 冲突与影响预检 | 网关/IP 错误会直接导致设备断网 | IP 重复、同网段、目标可达、引用设备和回滚计划 |
| 统一规则台账 | 迁移旧两张表且不丢离线、批量能力 | 按规则类型筛选、搜索、批量迁移/删除、冲突状态 |
| 策略优先级说明 | 未来全局、分组、设备规则会冲突 | 每项显示最终值、来源和“为什么生效” |

### 6.2 第二优先级

- 设备分组和批量套用规则，与 DHCP Policy Tag 完全分离；
- 定时联网和定时限速，例如儿童设备睡眠时段；
- 日/周/月流量历史及配额，优先以可选 `TrafficHistory` Adapter 接入；
- 首次发现、最近上线、长期离线清理和可控持久化；
- IPv6 静态保留及 IPv6 策略等价性；
- 安全的设备管理入口探测，明确 HTTP/HTTPS 与证书风险；
- 策略变更审计、最近修改人/时间、导出与恢复。

### 6.3 可选高级能力

- 每设备连接诊断、DNS 查询趋势与异常解释；
- 新设备上线通知和 Webhook；
- 来宾/IoT 网络分段与 Device Group 联动；
- 自动识别规则包更新，但必须本地执行、可回滚且不上传 MAC；
- 多 WAN/策略路由集成。它与 DHCP 网关分配不是同一个模块，不能混用术语。

## 7. 目标后端架构

继续坚持深模块。外部 Interface 保持少而稳定，UCI section、命令和插件差异封装在内部 Adapter。

### 7.1 `DeviceInventory` Module

Interface 仍以 `Snapshot` 为核心，增加 Device Alias 和持久/临时身份能力，但不吸收策略或遥测。设备类别错误只能影响呈现，不能改变 Device Identity。

### 7.2 `DeviceProfile` Module

Interface 只接受设备资料 patch：Unicode Device Alias、人工 Brand、人工 Device Category 和 Icon Preference。分组成员关系、DHCP Hostname 与所有网络策略不进入该 Module。自动图标只持久化用户选择“自动”，解析结果由读取模型按“手动图标 → 品牌+类型视觉 → 类型 → 通用电脑”计算。

### 7.3 `DeviceNetwork` Module

Interface 只提供 `Get`、`Plan`、`Apply` 三个操作，完成一个设备的地址模式、Address Reservation、ASCII DHCP Hostname 和 Internet Path。Implementation 将 Internet Path 一次编译为匹配的 DHCP 网关与 DNS 设置，隐藏 host/tag、option 3/6 和服务重载；同一计划不能留下“地址已改、路线未改”的半配置。

### 7.4 `DeviceRestrictions` Module

Interface 只提供 `Get`、`Plan`、`Apply`，协调 Internet Access Policy、Speed Policy、时段、额度和单设备例外。联网权限使用 firewall Adapter；限速优先使用 `nativeRateLimitProvider`，eqos 仅为固定 IPv4 兼容 Adapter，Bandix Adapter 仅作显式迁移输入。任一 Adapter 缺失只降低相应能力，不能拖垮另一能力或 Device Inventory。

### 7.5 `GatewayPolicy` Module

Interface 负责列出 Gateway Target、规划分配、读取引用和解释健康。Implementation 内部包含 DHCP Policy Tag、option 3/6、默认/本机/上级/旁路由、指定 IP 与 Floating Gateway Adapter；这些内部机制不得泄漏到普通 Interface。

### 7.6 `RouterContext` Module

Interface 按选定 LAN 返回拓扑证据、LAN 地址、上级网关、角色说明、DHCP authority 和路线可编辑性。DHCP authority 固定为 `local`、`external_observed`、`none_detected`、`ambiguous` 或 `error`；只有 `local` 且相关能力健康时允许修改设备 Internet Path，角色名称不能绕过这条规则。

### 7.7 `PolicyEffect` Module

Interface 聚合 Desired Policy、Applied Configuration 和 Observed Effect，供设备列表、详情、需要处理筛选与审计共同消费。它可以确认服务器配置和保存后的新租约，但在没有终端代理时不能声称终端实际采用了网关或 DNS。

### 7.8 `FloatingGateway` Module

当前写服务保留为配置 Adapter，上层新增小而完整的 Interface：

```go
type FloatingGateway interface {
    Get(ctx context.Context) (FloatingGatewayState, error)
    Plan(ctx context.Context, desired FloatingGatewayConfig) (ChangePlan, error)
    Apply(ctx context.Context, plan ChangePlan) (FloatingGatewayState, error)
    RunDrill(ctx context.Context, options DrillOptions) (DrillResult, error)
}
```

Module 负责翻译产品角色与插件 `main/fallback`、采集运行状态、双端一致性、预检和恢复。故障切换演练是显式高风险动作，不得自动执行。

### 7.9 `TrafficInsights` Module

`DeviceTraffic` 继续负责实时快照；跨重启历史、配额、连接和 DNS 由独立 `TrafficInsights` Interface 提供。这样可使用 Bandix Adapter，也可在不支持 eBPF 的设备上降级，不污染设备列表核心路径。

### 7.10 任务级写入事务 Seam

三个写入任务各自复用同一个内部流程：

```text
读取当前状态 → 生成计划 → 语义/冲突预检 → 快照
→ 原子写配置 → 重载最少服务 → 读取验证
→ 成功记录审计 / 失败恢复快照并再次验证
```

每个任务只能承诺自身范围内恢复。跨 dnsmasq、firewall、eqos 和 JSON 的故障通过补偿事务返回 `rolled_back` 或 `recovery_required`；不得宣称整个设备详情绝对原子。幂等键需使用持久状态版本或有界持久记录，不能只依赖进程内 map。

## 8. 前端实现原则

- 一个页面只有一个主要任务；全局设置不再嵌套另一组看不出层级的技术页签；
- 设备行只展示结果，编辑全部在详情或专门向导；
- 表单使用即时语义校验，网络写入前再由后端权威校验；
- 保存成功后展示“配置已保存”和“实际已生效”两个不同状态；
- 未安装依赖提供解释和安装入口，但用户关闭安装弹窗后仍能浏览其他设备信息；
- 危险操作确认必须包含对象、影响数量和恢复方式；
- 桌面、平板和 390 px 手机均保持一个主操作、无页面级横向滚动；
- 深色主题、键盘操作、屏幕阅读标签和中英文长文案属于完成条件；
- 新页面通过统一 store/composable 消费 Inventory、Policy、Gateway 和 Traffic 数据，禁止继续在组件内拼接多套 `any` 结构。
- `192.168.30.1` 是测试网络的关键主网关：自动化只能对它执行低频、串行、只读检查；负载、安装、重启、Apply、服务重载、故障注入和浮动网关演练必须默认拒绝。需要写入的端到端场景应使用隔离网关环境，不能以“可回滚”为理由影响基础网络。

## 9. 实施顺序

历史 M21～M40 已完成产品、领域、正式 UI 与发布基线；M41～M52 完成独立 Quickstart NetPolicy 引擎、显式 Bandix 迁移和四机真实吞吐验收。当前证据见 [M41～M52 验收记录](evidence/m41-m52/acceptance.md)。

## 10. 不可回归验收清单

每次产品、前端或后端改动至少核对：

- IPv4-only、IPv6-only、同 MAC 多地址、随机 MAC、离线重现和 IP 转移；
- 未命名、用户命名、原始厂商、人工类别和图标失败回退；
- 静态地址添加/编辑/删除/批量、离线规则和冲突；
- 跟随默认、本机、上级、旁路由、浮动网关和自定义网关路线；
- 全局限速关闭、插件未安装、设备限速、断网、恢复与规则保留；
- floatip 两个角色、对端不可达、互联网检测失败、接管、恢复和配置变化清理；
- 保存失败、重复提交、中途服务重载失败、回滚失败的可理解状态；
- 1440×900、1024×768、390×844，浅色/深色，中英文和键盘操作；
- 设备清单在任一可选插件失败时仍可浏览；
- 不上传 MAC，不引入来源/许可证不清的品牌 logo。

## 11. 证据入口

Quickstart：

- 页面结构：`web/src/pages/device/index.vue`
- 新设备中心：`web/src/pages/device/deviceCenterList.vue`
- 当前统一策略表单：`web/src/pages/device/components/devicePolicyPanel.vue`
- 经典设备/静态/限速/全局页面：`web/src/pages/device/deviceList.vue`、`staticStateList.vue`、`speedLimitList.vue`、`configure.vue`
- API 路由：`backend/modules/lancontrol/routes.go`
- 设备策略与事务：`backend/service/device_policy.go`、`device_policy_store.go`
- DHCP 与全局配置：`backend/service/lan_dhcp_usecase.go`、`lan_global_config_usecase.go`
- 浮动网关适配：`backend/service/lan_float_gateway_write_usecase.go`、`backend/modules/lancontrol/floatgateway/service.go`
- OpenWrt 打包：`/projects/workspace-linkease-ubuntu/openwrt-apps/istoreos-app-hub/apps/quickstart/quickstart/Makefile`、`luci-app-quickstart/Makefile`、`luasrc/view/quickstart/main.htm`

参考项目：

- `luci-app-oui/README.md`、`htdocs/luci-static/resources/oui/oui.js`
- `luci-app-bandix/README.zh.md`、`ARCHITECTURE.md` 及四个 LuCI view
- `floatip/floatip/files/floatip.config`、`floatip.sh`、`luci-app-floatip/luasrc/model/cbi/floatip.lua`

更细的逐文件事实、许可证和可借鉴模式记录在 [参考项目研究](lan-device-management-reference-research.md)。
