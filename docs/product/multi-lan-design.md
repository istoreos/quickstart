# 多局域网管理：产品与技术设计提案

状态：多 LAN 主体已实现，M01–M09 阶段验收通过，M10 最终验收未通过；不能视为已具备上线条件。2026-11-09 按用户反馈恢复桌面网络侧栏，以下导航口径已同步。依据当前 Quickstart 工作树，以及 `/projects/workspace-linkease-ubuntu/openwrt-apps/quickstart-netpolicy` 源码。后续按 [实施里程碑与验收计划](../archive/product/multi-lan-implementation-milestones.md) 推进，原型验证不代表业务实现完成。

## 实施前收敛：兼容优先，按需展示

本节为已认可的实施口径，优先于下文探索性原型的布局与可选扩展。当前实现遵循以下业务边界；桌面网络选择的视觉形式以本次侧栏要求为准。

### 产品范围

- 只有一个可管理 LAN 时，直接显示现有设备管理页，不显示网络选择、总览卡片或技术状态摘要。单一网络不一定叫 `lan`，也不一定使用 `br-lan`。
- 有多个独立 LAN 时，桌面左侧显示网络列表，手机使用紧凑网络选择；“全部局域网”用于设备浏览，选定一个 LAN 后管理其设备、分组和设置。复用当前三个页签，不另造一套管理界面。
- `br-lan`、firewall zone、引擎端口、revision 等信息仅供诊断使用，不成为普通用户必须理解或填写的字段。
- 首版不增加网络创建、VLAN 配置、跨 LAN 批量变更、自动策略迁移、网络重命名和新型管理台。网络配置继续由现有系统设置维护。
- 未知归属作为设备状态和列表筛选处理，首版不为它新增独立工作区。未确认网络归属时不能执行网络策略。
- 功能组件区分“已安装”和“当前网络可用”；不能因为本机已安装 NetPolicy，就宣称每个 LAN 已有流量控制能力。

### 2026-11-09：正式页面恢复网络侧栏

用户反馈正式页面与初始产品设计距离较大。本次将桌面恢复为左侧网络导航、右侧当前网络标题和三个业务页签；减少重复标题、外壳边框和分散的工具区。网络名称与网段来自同一网络清单，不新增设备计数请求、总览卡片、网络创建入口或技术诊断字段。单 LAN 隐藏网络导航，手机以紧凑选择器代替侧栏。预览与正式界面复用同一个页面组件，网络作用域与写入权限保持原有业务契约。

### 单 LAN 兼容约束

| 配置 / 客户端 | 约束 |
| --- | --- |
| 现有 `lan → br-lan` | 页面、已有策略、资料、分组、配额与历史继续可用；不要求迁移或重新启用服务 |
| 唯一网络为 `office → br-office` | 新前端自动选中该网络，页面仍按单 LAN 展示；请求显式指定该逻辑网络 |
| `lan/lan1/lan2` 多网络 | 新前端按选定网络提交请求，读写与缓存均在对应作用域内 |
| 没有 `lan` 的多网络 | 正常浏览与选择；旧接口缺少网络参数的写入返回错误，不猜测目标 |
| 旧前端 / 第三方调用缺少网络参数 | 保留默认 `lan` 语义；不能悄悄扩展为全网络读写 |
| 新前端连接旧后端 | 仅在网络清单接口明确不支持时退回原单 LAN 模式；登录失败、超时或服务异常不能伪装成“只有一个 LAN” |
| 同桥下的物理 `lan1/lan2` 端口 | 仍为一个 LAN，不因为端口名称而生成多网络工作区 |
| 同 MAC 出现在两个独立 LAN | 按网络分别观察和执行策略，不用全局 MAC 记录替代网络归属判断 |

旧默认 LAN 保留原有设备 ID、存储位置和 NetPolicy main 配置。新增 LAN 的资料、分组、配额、策略与历史使用隔离命名空间，优先通过同一存储实现的路径参数实现，不复制存储实现。已有全局 DHCP host/tag 的作用范围需要从实际配置解析；不能只改 Quickstart JSON 就宣称底层 dnsmasq 已隔离。

### 代码实施边界

1. 增加小型 LAN 作用域模块，负责列出可管理网络和解析一个选定网络，统一返回实际设备、全部前缀、DHCP 绑定与区域信息。复用已有 ubus/UCI 读取，首版不引入独立网络 UUID 注册平台。
2. 设备采集、合并和历史增加网络维度；保留现有设备标识格式，但目标始终由网络与设备共同确定。ARP/NDP、DHCPv4/v6、host hints、Wi-Fi 证据在合并前已关联网络。
3. 路由层统一解析网络参数并校验，业务模块通过实例依赖或显式参数获得作用域。禁止全局 `currentLAN`，禁止通过读取浏览器当前 URL 隐式决定后台任务的作用域。
4. DHCP、网关、地址保留、设备限制、分组、流量和配额复用原模块，通过作用域化依赖工作。模块缓存、存储路径、调度任务和事务日志必须同步隔离；不能只修改界面请求。
5. NetPolicy 保留每进程单接口结构，由 procd 命名实例提供隔离；旧 main 保持兼容。Quickstart 使用服务端配置选择实例，不让浏览器提供端口或服务地址。健康检查核对实际接口，不自动接管其他流量引擎的设备。
6. 所有共享 UCI 文件的写入、验证与回滚使用同一资源锁和配置版本约束。计划同时绑定网络配置与执行实例；切换网络、配置变化或实例重建后不得复用旧计划。
7. 前端增加一个统一的 LAN 上下文与请求入口，组件以该上下文调用现有业务接口；避免每个按钮重复拼接 network 参数。总览中的设备操作必须携带该行的网络，而非当前总览值。
8. 设计预览回到 `/devicemanagement` 的真实页面，通过多 LAN fixtures 展示已实施行为；独立 `/design/multi-lan` 仅作为历史提案参考，不能充当完成业务实现的证明。

### 实施顺序与完成标准

按“网络解析与契约 → 设备归属 → 系统写入与事务 → NetPolicy 实例 → 正式页面 → 回归与实机验证”依次实施。阶段可以分开提交，但不能将只读发现阶段对外描述为完整多 LAN 管理。

回归必须覆盖原单 LAN 全流程，以及双 LAN 的同 MAC、无默认 lan、自定义 DHCP section、共用 dnsmasq/zone、并发修改、回滚、切换网络的旧响应、实例失败/重启、IPv4/IPv6 与设备迁移。未完成隔离适配的功能须明确返回不可用，不能退回默认 LAN 执行；要提供完整多 LAN 能力，仍需完成相关适配与验证。

代码收敛标准：一套业务模块、一处网络解析、一套有明确网络身份的执行依赖；不复制整套页面或 ServiceBackend，不增加通用工作区框架。每项新抽象必须对应现有真实差异，测试通过实际业务接口检查跨网络影响，而非仅断言 URL 参数或模拟数据。

## 1. 建议结论

采用 **“全部局域网总览 + 单个 LAN 工作区”**，将 LAN 作为设备观察、网络设置与策略执行的明确作用域。

- 总览汇总各 LAN 的设备，列表显示网络归属；跨网络同 MAC 分别展示。
- 进入单个 LAN 后管理设备、分组与计划、局域网设置，复用现有三个页签。
- 每个 LAN 独立判断 DHCP 分配权、网关可编辑性、限速/配额可用性，不再共用默认 LAN 的能力结论。
- 网络自动发现与策略接管分开：发现 `lan1/lan2` 即可浏览；启用策略服务需要明确接入并验证。
- 首版利用 NetPolicy 单接口实例的现有实现，按实际接入设备运行隔离实例，由 Quickstart 在服务端选择实例；不直接将 MAC-only 的共享 map 挂到多个 LAN。

这里的 LAN 指 OpenWrt **逻辑网络**。`lan1/lan2` 若只是同一个 `br-lan` 中的 DSA 物理端口，则仍是一个 LAN，应作为接入端口展示；独立接口、独立网桥或 VLAN 子接口才可能构成不同网络。不能按名称前缀 `lan*` 发现网络。OpenWrt 的端口与网桥关系见 [官方 DSA 文档](https://openwrt.org/docs/guide-user/network/dsa/dsa-common-config-1-bridging-all-lan-ports)。

## 2. 当前问题与可复用基础

| 层次 | 当前代码证据 | 多 LAN 后果 / 处理方向 |
| --- | --- | --- |
| 旧设备发现 | `backend/service/network_device_list_usecase.go` 的 `ReadLANInterfaceName` 固定传入 `lan`，随后按该设备筛 ARP | 其他 LAN 的在线设备缺失；改为按发现的网络设备采集 |
| 新设备发现 | `backend/service/device_inventory_v2_source.go` 的 NDP、DHCPv6 固定 `br-lan`；DHCPv4 只读 `/tmp/dhcp.leases` | 部分外网段租约可能进入清单，但没有可信网络归属；需要多源、按网络归并 |
| 身份与归并 | `backend/service/device_inventory_v2.go` 以 `mac:...` 或 DUID/IAID 归并；`backend/models/device_inventory_response.go` 没有网络字段 | 跨 LAN 同 MAC、重复地址、IPv6 link-local 容易混合；归并键增加网络维度 |
| 旧连接字段 | `backend/service/lan_device_list_types.go` 的 `Intr: "lan"`，新模型的 `connection.kind` 表示有线/Wi-Fi | 连接方式和 LAN 归属必须分开，不能拿 `intr/kind` 充当 LAN ID |
| LAN 上下文 | `backend/service/router_context.go`、`router_context_backend.go` 已支持 `?lan=guest_20` | 可复用按 LAN 探测的接口；需补网络存在性、服务实例健康检查与绑定一致性 |
| 网络发现 | `backend/service/network_interface_inventory_store.go` 已读取 `network.interface dump`、firewall zone 绑定和端口关系 | 可复用读取逻辑；现有快照丢失部分前缀、运行设备与状态，需要补齐 |
| DHCP | `backend/service/lan_dhcp_settings.go`、`lan_dhcp_store.go` 读写 `network.lan/dhcp.lan` | 在 lan1 编辑也可能修改默认 LAN；必须先解析作用域，再构建写入计划 |
| 网关与地址保留 | `gateway_policy_store.go`、`device_network_policy_store.go`、`lan_static_assignment_write_store.go` 以默认 LAN、全局 MAC、全局 host/tag 工作 | DHCP host、tag、网关引用与冲突检查均需按真实作用域处理 |
| IPv4/IPv6 与流量 | `lan_stats_linux.go` 的 `readSystemLANPrefixes` 读默认 LAN；`device_traffic.go`、`traffic_insights*.go` 以设备 ID 保存统计和配额 | 前缀过滤、数据源选择、历史计数基线和配额都要增加网络与引擎身份 |
| 分组与计划 | `device_groups_store.go` 保存一套 `GlobalPolicy/Groups/DevicePolicies` | “全局策略”需明确定义为当前 LAN 默认策略；不同网络分组独立 |
| 限速适配 | `rate_limit_native_provider.go` 一个 `baseURL`、固定 owner instance，设备按 MAC 匹配 | 增加引擎注册表与按网络选择 provider；替换 owner 策略集时仅携带该网络的完整目标集合 |
| 旧防火墙路径 | `lan_speed_limit_write_store.go` 写死 `src=lan/dest=wan` | 不能仅替换 LAN 名；必须解析 firewall zone，且按接口限制实际生效范围 |
| 页面 | `web/src/pages/device/index.vue` 只有三个页签；`deviceInventory.ts` 无网络模型 | 新增工作区选择和网络上下文，逐页传递作用域 |
| NetPolicy 启动 | `packaging/quickstart-netpolicy.config/init` 一个 main、一个 interface、一个 port、一个 data_dir | 将服务改为命名实例；每实例独立设备、端口、目录与健康状态 |
| NetPolicy 契约 | `docs/interfaces/netpolicy-v1.md` 明确设备按 MAC 标识；`bandix/src/policy/mod.rs` health 返回一个 interface | v1 仅在已验证的单接口实例内部复用，不能当全局 MAC 契约使用 |
| NetPolicy 内核数据 | `bandix-ebpf/src/modules/traffic/maps.rs` 的流量、限速、配额、令牌桶均按 `[u8;6]` MAC 索引；`bandix/src/ebpf/shared.rs` 每次加载一个 eBPF 对象 | 不同 LAN 必须隔离对象与状态；未来共享 map 时才需要改变内核键 ABI |

## 3. 产品行为

### 3.1 导航与默认值

桌面左侧是“全部局域网 / 家庭 lan / 办公 lan1 / 访客 lan2 / 待确认归属”，右侧保留三个业务页签；移动端改为横向可滚动的网络选择。

初次访问：只有一个 LAN 时直接进入该 LAN，多个 LAN 时进入总览。明确 URL 中的 `network` 优先，其次恢复仍有效的上次选择；网络已删除则展示失效状态并返回总览，不静默切换到另一个可写 LAN。只有 `lan1/lan2`、完全没有 `lan` 也是正常配置。

名称显示“办公网络 · lan1”，并展示网段。业务名称是 Quickstart 的显示别名，修改别名不重命名系统网络。接口名不等于网桥名，也不等于 firewall zone 名。

### 3.2 全部局域网

- 顶部每个 LAN 一张摘要卡：名称、网段、设备数、策略状态；点击进入该 LAN。
- 设备表增加“所在网络”，保留搜索、在线状态与现有详情入口。总览数量按网络接入记录计数；不承诺物理设备去重。
- 首版总览提供浏览与单设备操作入口；设备详情明确显示归属与影响范围。网络级 DHCP、默认路线、分组计划需要选择 LAN 后编辑。
- 总览不提供跨 LAN 批量写入。将来若支持，必须分网络预览、逐网络返回结果，不能声称跨引擎原子提交。
- 无法归属的租约、手动记录或历史设备放入“待确认归属”，保留资料和历史；不默认归入 `lan`。

### 3.3 单 LAN 工作区

| 页签 | 行为 |
| --- | --- |
| 设备 | 仅当前网络的接入记录；详情显示网络、当前/历史地址、该网络内的限速/配额/地址保留/路线 |
| 分组与计划 | 只选择本 LAN 成员；路线目标、策略优先级和默认策略均在 LAN 内解释 |
| 局域网设置 | 当前 LAN 的 DHCP、默认路线、规则台账、功能组件状态；同时显示系统共用组件与本 LAN 接入状态 |

本机提供 DHCP 时才开放对应 DHCP/路线写入；外部 DHCP 时显示证据和操作指引，保持只读。NetPolicy 是否可用单独判断，外部 DHCP 不必然禁止限速，但流量必须实际经过支持的挂载点；未验证转发路径时不承诺策略生效。

浮动网关和遗留 provider 如尚未能按 LAN 隔离，显示“此网络暂不支持”及原因，不复用默认 LAN 的实例。网络间互访隔离继续由系统防火墙管理；限速和配额不表示访客网络已隔离。

### 3.4 变更与异常

- 所有变更预览显示网络、设备/分组、修改内容、受影响数量、相关共用服务。切换 LAN 清空设备选择、抽屉、未应用计划，并使旧请求失效。
- 表单修改后切换网络时，提示保留当前修改还是放弃；该提示属于产品交互，正式实现时补充。预览中切换直接重置示例草稿。
- lan1 引擎异常只锁定 lan1 对应写入；lan/lan2 的可用功能继续工作。服务未接入时可浏览设备，不显示虚构的 0 流量。
- 设备跨 LAN 移动后，原策略留在原 LAN，按当前网络默认策略处理；不把地址保留、路线、配额自动复制到新 LAN。人工明确迁移可在后续增加。
- 新发现网络先只读展示，不能在后台自动开启 DHCP、创建转发规则或抢占 Bandix 的接口。

## 4. 技术方案

### 4.1 LAN 作用域模块

建议新增 `LanScopeModule`，对调用者仅暴露 `List` 与 `Resolve(networkId)`。内部统一完成发现、逻辑网络到设备映射、DHCP section/服务实例解析、zone 绑定、接入能力与版本计算，避免各写入模块重复拼接 `dhcp.<networkId>`。

建议模型（新增接口设计，非现有字段）：

```ts
type LanScope = {
  networkId: string             // 首版为 UCI 逻辑接口 section 名，如 lan1
  generation: string            // 网络被删除重建后变更，防止旧计划套用到新网络
  displayName: string
  device: string                // 实际 ARP/NDP/策略接入设备，例 br-office 或 br-lan.10
  aliases: string[]             // 同一链路的逻辑别名，不重复挂载和计数
  ipv4Prefixes: string[]
  ipv6Prefixes: string[]
  firewallZones: string[]
  dhcpBindings: Array<{ section: string; instance: string; leaseFile?: string }>
  state: 'up' | 'down' | 'unknown'
  capabilities: Record<string, { state: string; reason?: string }>
  version: string
}
type DeviceTarget = { networkId: string; deviceId: string }
```

发现来源：UCI `network/firewall/dhcp/wireless` + ubus dump/runtime，复用现有网络 inventory reader；由配置和运行信息综合识别 LAN 候选，不用名称前缀或“有无默认路由”作为唯一标准。排除已明确为 WAN、loopback、容器、隧道的网络；自定义/不明确用途列为候选，供用户明确纳入。停用 LAN 保留已保存信息与诊断，不执行写入。

同一实际设备上的逻辑别名归为一个链路工作区或明确标记无法独立管理，不能为每个别名重复挂载引擎。VLAN 父桥与子接口不能同时重复监控同一份流量；接入解析必须选择经过验证且不重叠的挂载点。桥接转发、硬件交换与 offload 的可见性必须实机验证，发现设备不等于拥有完整流量控制能力。

`generation` 需要持久注册表：记录纳管关系、绑定变化与删除事件。接口改名按新网络处理；移除旧映射，保留原策略待人工迁移。设备名/ifindex 是运行事实，不是持久网络主键。

### 4.2 设备发现与身份

保留现有 `deviceId` 的不透明字符串格式，新增 `networkId`，把 **`(networkId, deviceId)` 作为接入记录与策略目标的唯一键**。浏览器 row key、选中项、缓存、去重、历史、流量和计划引用都使用这一组合；不可先按 MAC 全局合并后再筛 LAN。

ARP/NDP observation、每个地址、DHCPv6 DUID/IAID、host hint、Wi-Fi 关联信息都携带网络/设备/证据来源；IPv6 link-local 还需链路 zone。不同 LAN 的同 MAC 或同 DUID/IAID 分开；相同 MAC 不作为同一物理设备的证明。资料覆盖存储也需按网络解析，保留旧资料仅作为默认 LAN 的迁移来源。

采集可一次读全表再按已解析设备分组，避免每 LAN 反复执行 ubus 和命令。DHCPv4 遍历实际 dnsmasq 实例及 leasefile；租约本身没有可靠接口信息时，结合实例绑定、前缀、邻居表归属。重叠前缀、共用租约文件、多个可匹配网络时保留“待确认”，禁止猜测。离线归属只代表上次确认网络，不能当作当前在线证据。

普通多网段是首版范围；不同 VRF/netns 的完全重叠地址不宣称首版支持。遇到不足以区分的重叠网络，降级只读并给出原因。

### 4.3 HTTP 契约与写入安全

沿用 `/lanctrl/v2/`，新增/扩展的具体路径须同步路由、模型、YAML 和契约样例：

```text
GET  /lanctrl/v2/networks/
GET  /lanctrl/v2/devices/?network=all|lan1
GET  /lanctrl/v2/router-context/?lan=lan1    # 现有参数可复用
GET  现有 DHCP / 路线 / 分组 / 流量 / 策略读取接口 + ?network=lan1
POST 现有 plan/apply 接口 + body.networkId = "lan1"
```

- 总览 `all` 仅用于读取；写入必须是一个明确 LAN，`networkId` 和设备网络关系由服务端重新检查，不能信任前端过滤结果。
- 计划保存 networkId、generation、拓扑绑定版本、配置版本、目标设备、引擎 ID、引擎 revision。应用时核对；未知网络、跨网络目标、删除/重建/移动、计划过期都拒绝，不能 fallback 到默认 LAN。
- 计划和幂等键、事务日志、审计、回滚状态增加 networkId。网络上下文作为请求/模块参数传递，不修改服务级可变 `currentLAN`，避免并发请求串网。
- 老接口不带 network 的读取仍保持既有默认 LAN 语义；正式多 LAN 前端必须显式传参。旧写入仅可指向有效、无歧义的旧默认 LAN；不存在 `lan` 则返回明确错误，绝不自动写入 lan1。提供契约版本/能力字段避免旧前端误写。
- 前端通过统一 network context/composable 生成请求，切换网络取消旧请求，并用请求代次阻止旧响应覆盖当前页面。

### 4.4 DHCP、网关和共用配置

DHCP section 不必与逻辑接口同名，要按 `option interface` 和实例绑定解析；处理多条绑定的冲突，而非简单将字符串 `lan` 换成 `lan1`。

地址池校验只针对当前网络的前缀、路由器地址、网关、浮动地址和保留地址，但实际 UCI 文件与 dnsmasq 实例可能共用。`config host` 也不是天然按 LAN 隔离：`instance` 和 range/tag 机制需要由适配模块编译并预检实际 dnsmasq 配置，不能凭空给 host 加 `interface=lan1` 就声称隔离完成。目标固件无法表达可靠作用域时，对冲突场景拒绝写入，尤其同 MAC 的不同路线/不同地址。相关配置语义见 [OpenWrt DHCP 文档](https://openwrt.org/docs/guide-user/base-system/dhcp)。

DHCP tag、自动生成 section、路线 ID、地址保留项使用网络命名空间；现有按 MAC 查找所有 host 并整体替换的逻辑需要改造。按真实 dnsmasq 服务范围执行 reload/verify，并在预览显示其他共用该服务的 LAN。

**网络作用域与文件事务锁不能混为一谈。** 即使请求修改不同 LAN，也会写同一个 `/etc/config/dhcp`，因此仍需配置文件/服务级写锁，重读版本后定向修改。回滚须在同一锁/版本约束下恢复，不能用陈旧整文件快照覆盖其他 LAN 后续成功变更。复用并扩展 `task_transaction*.go`，继续遵守现有任务事务设计。

firewall zone 通过配置解析，不能使用 networkId 代替 zone。多个 LAN 共用一个 zone 时，策略还需按实际接口限制；provider 做不到则不可用，不能用整 zone 的规则误伤其他 LAN。

### 4.5 NetPolicy：推荐先做隔离实例

当前 Rust 实例、引擎事务、eBPF maps 都是单接口模型，先通过 procd 命名实例复用这一事实：

```text
Quickstart LanScope + EngineRegistry
  ├─ lan  → br-lan    → loopback endpoint A → 独立 data_dir / eBPF / revision
  └─ lan1 → br-office → loopback endpoint B → 独立 data_dir / eBPF / revision
```

每个实例依然在本实例内以 MAC 为键；Quickstart 的网络选择确保相同 MAC 不进入同一错误实例。将来可实现“一个进程管理多个独立 context”，保持上层路由接口不变。本轮不建议共享一套 MAC map；若未来要共享，应同步升级 `{scopeId, mac}` 内核键、用户态 ABI、持久化和版本契约。

| NetPolicy 文件 | 建议改动 |
| --- | --- |
| `packaging/quickstart-netpolicy.config` | 从单 main 扩展为按 network 命名的实例；保存 network/interface/port/data_dir 与 enabled；保留旧 main 兼容迁移 |
| `packaging/quickstart-netpolicy.init` | 枚举实例、命名 `procd_open_instance`，校验重复设备/端口/目录，生成每实例命令；reload 不应无条件重启全部健康网络 |
| `bandix/src/command.rs` | 保留每进程一个 `--iface`；增加可选 network/instance 标识及健康输出，按实例处理重启与前缀变化；无需首版改成多 `--iface` |
| `bandix/src/web.rs` | 本地控制端点明确限制 loopback，不能依赖 debug/release 隐式选择监听地址；若未来换 Unix socket 则另做版本设计 |
| `bandix/src/engine_guard.rs` | 继续按实际设备互斥；纳入实例标识、接口消失/重建检测和冲突诊断；验证多实例不会错误释放别人的 lease/TC link |
| `bandix/src/policy/mod.rs`、`transaction.rs` | health 增加实例/网络/运行代次；计划绑定实例并验证；每实例保持原有 revision/owner 替换语义 |
| `bandix/src/policy/store.rs`、`storage/{traffic,quota,hostname}.rs` | 验证所有文件基于独立 data_dir；策略、配额、历史和计数基线不能共享文件；旧实例迁移仅一次 |
| `bandix/src/ebpf/shared.rs`、`monitor/traffic.rs`、`device.rs` | 验证每实例拥有自己的对象、设备扫描、邻居、前缀；TC 清理只处理自己持有的 link；全机本地前缀与按网络计数规则需明确 |
| `bandix-ebpf/src/modules/traffic/{maps,mod}.rs`、`bandix-common/src/lib.rs` | 隔离实例路线下首版无需强制改 MAC key ABI；仍需验证重复挂载、跨 LAN 流量分类和 IPv6/offload 行为 |
| `docs/interfaces/netpolicy-v1.md` | 补充“实例绑定的网络上下文”与发现/health 契约；若新增必填字段则做版本协商，不能无版本改变旧 v1 |
| `scripts/smoke/{topology-smoke,ipv6-topology-smoke,resource-check}.sh` | 扩展双 LAN、同 MAC、单实例重启/失败、热插拔与资源增长验证 |

EngineRegistry 建议由 Quickstart 的本机配置和 NetPolicy 实例配置构建，逐实例 health 验证 network/interface/instance。浏览器仅提交 networkId，不接受浏览器传入任意 baseURL、端口或设备名。现有 `8765` 可作为旧 lan 实例保留，新端口由持久配置分配并检查占用；不能按列表下标动态漂移。

每个实例的 owner desired-set 替换只能携带该实例的策略，避免在修改一台 lan1 设备时清空该 LAN 其他策略，或把 lan 策略下发到 lan1。引擎重启/计数归零要用 instance epoch 重设基线，防止配额累计跳变。

流量口径：设备“互联网流量”应排除已知本地 LAN 前缀；lan→lan1 的内部访问在两侧都可能被观察到，不能把两侧流量简单相加为路由器公网总量。优先复用 WAN 汇总指标，总览按 LAN 展示设备流量，不提供未经校正的跨 LAN 总和。路由模式、旁路模式、桥接模式和软/硬 offload 分别验证。

单实例独立 eBPF 对象会增加内存与进程开销；需要在目标机测出 1/2/4 个 LAN 的资源曲线和容量，再公布支持上限。现有 1024 项是单实例契约上限，不代表路由器能无条件运行 N×1024 个终端。

## 5. Quickstart 代码改动清单

| 改动组 | 现有文件 / 建议新增 | 重点 |
| --- | --- | --- |
| LAN 清单与作用域 | 新增 `backend/models/lan_scope.go`、`backend/service/lan_scope*.go`；复用 `network_interface_inventory_*`、`backend/ubus/network_device.go` | 统一解析逻辑网络、设备、前缀、DHCP、zone、版本、能力 |
| 路由与契约 | `backend/modules/lancontrol/routes.go`、`backend/service/lan_control_backend.go`、`router_context_backend.go`、`backend/yamls/lanctrl.yaml`、`backend/models/*` | LAN 读取接口和所有相关请求/响应携带作用域；旧契约兼容 |
| 设备清单 | `network_device_list_usecase.go`、`lan_device_list_{inventory,types}.go`、`device_inventory_v2{,_source}.go`、对应模型 | 多设备采集、按网络归并、归属证据和未知状态；离线/手动记录迁移 |
| DHCP / 静态地址 | `lan_dhcp_{settings,store,types}.go`、`lan_static_assignment_write_*`、`lan_global_config_*` | 避免默认 LAN 硬编码、解析 DHCP section、作用域化 host/tag/冲突校验 |
| 路线与浮动网关 | `gateway_policy*.go`、`device_network_policy*.go`、`floating_gateway*.go`、`lan_float_gateway_write_*` | 路线归属、引用、前缀、DHCP 权限、服务实例能力；不支持处明确只读 |
| 策略执行 | `device_policy*.go`、`rate_limit*.go`、`lan_speed_limit_write_store.go`，新增引擎注册适配模块 | 网络目标、provider 选择、实例 health、完整目标集合并、zone/接口精确匹配 |
| 流量 / 配额 | `lan_stats*.go`、`device_traffic.go`、`traffic_insights*.go` | 按网络取样、统计/基线/配额存储、引擎重启、跨 LAN 分类 |
| 分组与计划 | `device_groups*.go`、`backend/models/device_groups.go` | 每 LAN 默认策略、成员和日程；拒绝跨 LAN 引用；存储 schema 迁移 |
| 事务 / 审计 / 接管 | `task_transaction*.go`、`network_audit.go`、`lan_device_migration*.go`、`rate_limit_migration.go`、`network_rules*.go` | networkId/generation、共用文件锁、迁移预览、旧规则归属和网络过滤 |
| 前端上下文 | `web/src/pages/device/index.vue`；新增 `lanScope.ts/useLanScope.ts`、网络选择展示模块 | URL 与作用域、布局、请求取消、草稿和抽屉生命周期 |
| 前端业务面板 | `deviceCenterList.vue`、`deviceInventory.ts`、`devicePolicy.ts`、`deviceGroupsPanel.vue`、`lanSettingsPanel.vue`、`lanDhcpSettings.ts`、`gatewayTargets.ts`、`components/*` | 网络列、组合 row key、作用域请求、每 LAN 能力、影响预览、批量选择限制 |
| 首页与旧入口 | `web/src/home/networkBusiness.ts`、首页设备摘要、`plugins/store` 及旧设备读取入口 | 设备汇总说明与跳转网络；不能通过旧 store 把总览又缩为默认 LAN |
| 设计与测试样例 | `web/src/design/*Fixtures*`、`docs/contracts/`、后端/前端测试、`scripts/ops/lan-device-*` | 多 LAN 数据、边界契约与实机验证 |

旧首页/新手向导中的 `lan` 不能无差别全局替换。新手向导若本来只配置系统主 LAN，保留其产品语义；只在它进入多 LAN 管理流程时明确跳转作用域。

## 6. 迁移与交付顺序

1. **P0：多 LAN 只读闭环。** LAN 清单、设备归属、IPv4/IPv6 采集、总览/切换、按 LAN 上下文和健康状态；`lan1/lan2` 没有默认 `lan` 也能正常打开。没有隔离写入能力时按功能只读。
2. **P1：单 LAN 写入闭环。** NetPolicy 多实例接入、限速/配额、DHCP/地址保留/路线作用域、事务与旧数据迁移。只有整条路径验证通过才为该 LAN 开放对应能力。
3. **P2：分组计划与完善。** 分组/默认策略、规则台账、浮动网关支持与限制、首页跳转、资源与性能、诊断与恢复；若首版承诺完整多 LAN，上述相关能力必须完成或明确按功能锁定。

迁移前备份现有 JSON、UCI 和引擎 data_dir；新增 schemaVersion 与迁移标记。旧数据只在可证明属于默认 LAN 时迁入该作用域；多义 host/tag/同 MAC/旧流量历史不复制给每个 LAN，保留待确认或明确标记为旧作用域历史。旧 NetPolicy main 的数据只映射到原实际接口，新增实例使用独立空目录，不直接复制旧配额计数。

新固件/后端对旧前端保持契约兼容。降级时不得让旧版本把新格式误读为默认 LAN 并覆盖多网络配置；需要备份恢复或阻止不兼容写入。网络热插拔重建时先验证配置与实例 generation，再恢复策略，不只看同名设备。

## 7. 验收要求

- 普通 `lan + lan1 + lan2`，以及只有 `lan1 + lan2`；同桥物理端口不会错误生成独立网络。
- 自定义名称 `office/guest/iot`、不同 firewall zone，以及多个 LAN 共用同 zone。
- 两个独立桥、VLAN 子接口、同桥别名；不在父桥/子接口上重复统计和执行策略。
- 同 MAC、同 IP、同 DUID/IAID 在不同 LAN 的观察、地址与策略不串网；IPv6 link-local 不跨链路合并。
- DHCP section 名与网络名不同；单/多 dnsmasq、自定义 leasefile、外部 DHCP、歧义归属。
- 修改 lan1 的池、tag、路线、地址保留或限速，只改变该目标；lan 中同 MAC 的策略保持原值。
- 相同服务/文件上的并发修改和失败回滚，不覆盖另一 LAN 的成功变更；客户端伪造跨网络目标被拒绝。
- 快速切换 LAN 后旧请求、旧选择、旧计划不可误应用；网络删除重建后旧计划拒绝。
- lan1 引擎失败/重启、端口占用、接口重建、Bandix 争用时只隔离对应能力，其他网络保持可用。
- 设备跨网络后不自动继承原策略；旧地址/配额不泄漏到新网络。
- IPv4/IPv6 下载与上传、配额耗尽、跨 LAN 访问、软/硬 offload、路由/旁路路径实测；没有经过挂载点的流量不假报有效。
- 1/2/4 个 LAN 的 CPU、RSS、BPF map 内存、采样延迟与闪存写入评估，明确设备容量。

## 8. 设计预览与本次范围

```sh
./scripts/ui-design.sh start
# http://127.0.0.1:5180/design/multi-lan
```

独立 Vue 文件：`web/src/design/MultiLanPreview.vue`，仅由 `web/src/design/mount.ts` 注册。支持网络切换、设备搜索、在线筛选、网络归属、详情与限速预览、DHCP 修改预览、网络内分组入口，以及“无默认 lan / 单 LAN 服务异常”场景；窄屏可使用。

预览全部使用合成数据，确认只更新本页示例。现有三个网络名称/地址仅为说明；预览未实现真实保存、DHCP 地址校验、纳管操作与网络生命周期，也未证明实机多 LAN 生效。正式代码与外部 NetPolicy 源码本次未改动。

本次验证：`npm --prefix web run tsc`、独立设计构建、`multi-lan-review.cjs` 的浏览器交互检查通过。检查覆盖跨网同 MAC 分列、lan1 限速演示不改变 lan2、外部 DHCP 只读、DHCP 示例按网络保存、无默认 lan、单网络引擎异常、未知归属锁定及 390px 布局；无浏览器脚本异常、无路由器或外部网络请求。设计构建产物位于 SSD 缓存，未写入生产 dist。另修正了设计配置中与当前 Vite 不兼容的 HTML 钩子格式，仅影响预览工具。

截图：[桌面总览](ui-design/multi-lan/overview.png)、[lan1 设置](ui-design/multi-lan/lan1-settings.png)、[移动端](ui-design/multi-lan/mobile.png)、[移动端设置](ui-design/multi-lan/mobile-settings.png)、[深色主题](ui-design/multi-lan/dark.png)。[检查结果](ui-design/multi-lan/review.json)仅证明设计交互，不替代上述实机验收。
