# DHCP Hostname 安全与设备统计性能研究

> 调研日期：2026-09-24  
> 范围：Quickstart Go 后端与设备中心、OpenWrt 24.10/dnsmasq 2.90、`luci-app-oui`、`luci-app-bandix`  
> 方法：以仓库源码、上游源码/RFC 和测试机只读检查为准；推断与建议单独标注。

## 1. 结论

1. **当前测试固件不支持中文 DHCP Hostname。** 测试机的 dnsmasq 2.90 编译为 `no-i18n no-IDN`；同一进程的 `--test` 明确拒绝中文并报告 `bad DHCP host name`，ASCII 名称通过。
2. **中文应该进入“设备备注名（Device Alias）”，绝不能自动写入 DHCP Hostname。** Alias 是 Quickstart 自己的 UTF-8 展示字段；DHCP Hostname 是会进入 dnsmasq/DNS 的协议字段，必须严格校验。
3. **现有后端存在比中文更严重的输入边界问题。** `Hostname`、MAC、IP、Tag 被拼入单引号包裹的 shell 命令，然后由 Bash 执行；后端没有权威 Hostname 校验。仅靠旧前端移除部分中文和空白，既不能保证 dnsmasq 可接受，也不能抵御引号等字符。
4. **当前实时统计已有正确的“按需启动、容量上限、TTL、短环形历史”基础，但流量接口每 3 秒重做完整设备清单，并每次写 `/tmp` 历史文件；`DeviceTraffic.totals` 还没有淘汰。** 这是后续增长前必须先修的主要风险。
5. **conntrack 采样的峰值与连接数线性增长。** 当前每次获取完整 conntrack，再复制为第二个 flow slice；在连接数很高的路由器上，短时分配和 GC 压力会显著高于设备数量本身。
6. **Bandix 应作为可选 TrafficInsights Adapter，而不是默认替换。** 其价值是历史、连接、DNS、配额；代价是额外守护进程、eBPF maps、持久化和 Linux 6.x/接口/硬件卸载限制。

## 2. 三种“名称”必须分开

| 名称 | 用户语义 | 字符集 | 存储/使用位置 | 是否下发到 DHCP/DNS |
|---|---|---|---|---|
| 设备备注名 `Device Alias` | 用户给设备起的名字，如“客厅电视” | UTF-8，可中文 | Quickstart 自有持久化 | 否 |
| DHCP Hostname | 局域网协议主机名，如 `living-room-tv` | 严格 ASCII LDH | UCI `dhcp host.name` / dnsmasq | 是 |
| 发现名称 `Observed Hostname` | 客户端上报或租约里观察到的名称 | 不可信输入，只读展示 | DHCP lease/host hints/inventory | 不应未经校验回写 |

产品规则：

- 默认只展示并编辑“设备备注名”；支持中文、日文、Emoji 等正常 UTF-8 文本。
- DHCP Hostname 放进“高级网络设置”，说明它会成为局域网 DNS 名称。
- 备注名为空时可以回退显示发现名称，但两者不能共用写入字段。
- 不得把 `displayName` 自动填入 DHCP Hostname。
- 非法 DHCP Hostname 必须明确拒绝并解释，不要静默删除字符后保存成另一个名字。

## 3. 协议与 dnsmasq 的明确边界

### 3.1 RFC 边界

- RFC 2132 §3.14 的 DHCP Host Name Option 引用 RFC 1035 的字符限制。
- RFC 952 的主机标签使用字母、数字和连字符；RFC 1123 §2.1 放宽为首字符也可以是数字。
- 每个 DNS label 最长 63 个字符；主机标签不能以连字符开头或结尾。
- 因而中文原文不是传统 DHCP Hostname 的可移植表示。IDN/Punycode 是另一个明确能力，不能假设每台路由器的 dnsmasq 都编译了支持。

建议后端采用单标签、最小可移植规则：

```text
空字符串，或：
^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$
```

- 长度按 ASCII 字节计，1～63。
- 不接受空格、点、下划线、中文、控制字符、引号和 shell 元字符。
- dnsmasq 2.90 的实现接受非首位下划线，但 RFC 主机名兼容性更窄；产品不应依赖此扩展。
- 若未来支持 FQDN/IDN，必须作为独立能力探测和版本化契约实施，不能放宽现有字段。

来源：

- [RFC 2132 §3.14](https://www.rfc-editor.org/rfc/rfc2132.html#section-3.14)
- [RFC 952](https://www.rfc-editor.org/rfc/rfc952.html)
- [RFC 1123 §2.1](https://www.rfc-editor.org/rfc/rfc1123.html#section-2.1)
- [dnsmasq 官方手册 `--dhcp-host`](https://thekelleys.org.uk/dnsmasq/docs/dnsmasq-man.html)

### 3.2 dnsmasq 2.90 源码结论

官方 dnsmasq 2.90 源码中：

- `src/util.c::check_name()` 在未编译 `HAVE_IDN`/`HAVE_LIBIDN2` 时直接拒绝非 ASCII 字节。
- `src/util.c::legal_hostname()` 对第一段名称逐字符限制为字母、数字，以及非首位 `-`/`_`。
- `src/option.c` 解析 `--dhcp-host` 时先 canonicalise，再调用 `legal_hostname()`；失败返回 `bad DHCP host name`。

来源：[dnsmasq 2.90 官方发布包](https://thekelleys.org.uk/dnsmasq/dnsmasq-2.90.tar.xz)，文件 `src/util.c`、`src/option.c`。

### 3.3 测试机直接验证

在 `192.168.9.215` 做了只读检查，没有改动 UCI 或服务状态：

```text
Dnsmasq version 2.90
Compile time options: ... no-i18n no-IDN ... UBus ... DHCP ...

dhcp-host=AA:BB:CC:DD:EE:FF,测试设备,192.168.9.99
=> dnsmasq: bad DHCP host name at line 2 of stdin

dhcp-host=AA:BB:CC:DD:EE:FF,test-device,192.168.9.99
=> dnsmasq: syntax check OK.
```

验证命令使用 `dnsmasq --test --conf-file=-` 从标准输入检查临时文本，没有写配置文件，也没有重启 dnsmasq。

## 4. 当前 Hostname 写入链路的具体风险

### 4.1 前端校验不一致

- 旧页面 `web/src/pages/device/deviceList.vue` 与 `staticStateList.vue` 只用 `/[\u4e00-\u9fa5\s]/g` 删除部分汉字和空白。
- 该规则漏掉扩展汉字、日文、韩文、Emoji、标点、引号、反斜杠和 shell 元字符。
- 新页面 `web/src/pages/device/components/devicePolicyPanel.vue` 没有 Hostname 字符校验。
- 新页面 `syncForms()` 使用 `policy.static.hostname || props.device.displayName`，未来一旦 `displayName` 成为中文 Alias，就可能把中文送入 Hostname 写接口。

### 4.2 后端没有权威校验

- `backend/service/lan_static_assignment_write_usecase.go` 仅验证 MAC 非空和 action 枚举，没有校验 Hostname、MAC、IP 的语法。
- `backend/modules/lancontrol/staticassignment/service.go::BuildCommands()` 直接将输入插入 `uci set ...='%s'`。
- `backend/utils/batch_command.go` 最终以 Bash 执行拼接命令。
- 因而单引号等输入不仅可能生成坏 UCI/dnsmasq 配置，也形成命令注入边界；前端过滤不是安全边界。
- `backend/utils/uci.go::UciCommitAndApply()` 先 commit，再发 `config.change`；OpenWrt init 脚本把 UCI 名称写成 `--dhcp-host=...` 并以该配置启动 dnsmasq。坏 Hostname 可能令新进程退出，影响 DHCP/DNS 可用性。

OpenWrt 生成与启动路径来源：

- [OpenWrt 24.10 `dnsmasq.init`](https://github.com/openwrt/openwrt/blob/openwrt-24.10/package/network/services/dnsmasq/files/dnsmasq.init)，`dhcp_host_add()` 与 `procd_set_param command`。

### 4.3 必须实施的安全顺序

1. API DTO 将 `alias` 与 `dhcpHostname` 分字段；删除自动复制。
2. Go 后端执行上述严格 Hostname 校验；前端只做即时提示。
3. MAC、IPv4、Tag/路线 ID 同样做类型化校验。
4. 不再拼 shell；优先使用 UCI 库，无法避免命令时使用参数数组和集中转义 Adapter。
5. 写入前生成候选配置并运行 `dnsmasq --test`。
6. 应用后验证 dnsmasq 实例仍为 running；失败恢复备份并再次验证。
7. legacy 与 v2 端点必须经过同一个 Validator/Transaction，不能只修新页面。

## 5. 当前实时统计实现

### 5.1 已经做对的部分

| 机制 | 当前实现 | 评价 |
|---|---|---|
| 采样周期 | `lanStatTimeTick = 3s` | 家用实时展示合理 |
| 空闲停止 | 无请求约 30s 后停止、关闭 conntrack socket | 正确，避免后台永久采样 |
| 页面隐藏 | 前端停止 timer，恢复可见后再拉取 | 正确 |
| 失败退避 | 3s 指数退避，最高 30s | 正确 |
| 主机状态 | 最大 2048，空闲 TTL 10min | 有硬上限 |
| 每主机短历史 | 12 个采样槽 | 有硬上限，约 36s |
| 地址归属状态 | 最大 2048，TTL 10min | 有硬上限 |
| 浏览器趋势 | 每设备最后 20 点 | 有硬上限 |
| LAN 前缀 | 每分钟刷新，失败保留上次结果 | 正确 |

证据：`backend/service/lan_stats_linux.go`、`device_traffic.go`、`web/src/pages/device/deviceCenterList.vue`、`deviceTelemetry.ts`。

### 5.2 P0 性能风险

#### A. 每次流量请求都重做完整 Inventory

`DeviceTrafficModule.Snapshot()` 首先调用 `inventory.Snapshot()`；设备中心成功时每 3 秒调用一次流量接口。因此每个可见浏览器都会反复执行：

- ARP/设备读取；
- DHCPv4、NDP、DHCPv6 读取；
- `luci-rpc getHostHints`；
- LAN prefix ubus；
- Wi-Fi 关联读取；
- 聚合、分类、排序；
- JSON marshal 并写入、rename `/tmp/quickstart-device-inventory-v2.json`。

这使实时流量的成本错误地包含完整设备发现与磁盘/RAMFS写入。多浏览器会放大 Inventory 成本。

#### B. `totals` 没有淘汰

`DeviceTrafficModule.addresses` 有 TTL/上限，但 `totals map[deviceID]` 没有 TTL、容量限制或删除。随机 MAC、短命身份或设备 churn 会让它在进程生命周期内单调增长。现有“两小时 20 设备”测试只检查 `addresses` 和总 heap 增长，没有覆盖不断更换 DeviceID 的场景。

#### C. conntrack 完整复制带来峰值

- Netlink 路径先由库 `Dump()` 得到完整 `[]conntrack.Flow`，再构造完整 `[]lanTrafficFlow`。
- `/proc/net/nf_conntrack` fallback 也先把所有行解析进 slice，再聚合。
- 每 3 秒时间复杂度为 O(F)，峰值额外内存也为 O(F)，F 是 conntrack 数量，不是设备数。
- 测试机只读快照为 25～42 条，而 `nf_conntrack_max=65535`；当前安静测试值不能代表压力上限。
- Netlink 一旦失败，`procFallback` 在本进程余下生命周期保持 true，不会尝试恢复低开销路径。

#### D. 满容量 churn 的淘汰复杂度

- `LanStats.addSpeeds()` 对新主机先全表 prune，满时再全表找 oldest。
- `DeviceTraffic.enforceAddressLimit()` 每超出一个元素都全表找 oldest。
- 常态是 O(H)，但在恶意或异常地址 churn 下可能形成 O(new × H)。

#### E. 新页面仍依赖 legacy 聚合

设备中心初次加载并行请求 v2 Inventory 和 legacy `listDevices`。legacy 流量增强按设备逐个通过 channel 调用 `reqHosts()`，产生 O(D) 次串行请求和分配。应由一个 v2 snapshot 返回必要摘要，移除新页面的 legacy 数据依赖。

### 5.3 并发和持久化事实

- `LanStats` 使用单 goroutine 和 channel 串行拥有状态，避免 map 并发写，但所有请求也共享这一队列。
- `DeviceTrafficModule` 用 mutex 保护归属与 totals；锁覆盖 hosts map 构造、所有设备归属和 response 构造。
- `DeviceInventoryModule` 的 mutex 覆盖历史合并、分类、排序和历史写入；慢写会串行阻塞其他 Inventory 调用。
- Inventory 历史位于 `/tmp`，重启后按 boot ID 丢弃，不磨损 flash；但每 3 秒 marshal/write/rename 仍消耗 CPU、分配与 I/O。
- 分类 override 持久化有 2048 上限并按需加载，风险低于实时路径。

测试机单点观测：Quickstart `VmRSS` 约 22 MiB、8 threads、Inventory JSON 约 6.8 KiB。它仅是当前 x86/低负载快照，不能作为低内存路由器的容量证明。

## 6. 性能整改建议与验收门槛

### 6.1 必须先做的结构调整

1. `DeviceInventory` 变成共享缓存：后台或 single-flight 刷新，流量接口只读最近 snapshot。
2. Inventory 的正常刷新不快于 10 秒；策略写入/租约事件可以主动失效缓存。
3. 历史文件仅在语义内容变化时写，并做 30 秒 debounce；实时流量读取不得触发持久化。
4. `totals` 增加 `LastSeen`、TTL 和与 Inventory 相同的硬上限；设备消失后可按产品定义保留一段时间，但不能无限增长。
5. conntrack 直接流式聚合到 host map，去掉第二份完整 flow slice；proc fallback 必须逐行聚合。
6. 满容量淘汰改成一次批量排序/heap/LRU，避免每个新地址全表扫描。
7. 新设备中心停止请求 legacy `listDevices`；所需策略摘要由 v2 API/缓存提供。
8. 所有采样上限、TTL、采样周期在一个配置结构集中定义，并暴露只读诊断数据。

### 6.2 可自动验收的性能标准

- 同时打开 1、3、10 个设备页面，conntrack 实际采样仍最多每 3 秒一次，Inventory 全源刷新仍最多每 10 秒一次。
- 页面全部隐藏/关闭后 35 秒内停止采样，conntrack socket 关闭。
- 运行 24 小时 DeviceID churn 测试后，hosts、addresses、totals、history 四类容器均不超过声明上限，GC 后 heap 不呈线性增长。
- 1k、10k、`nf_conntrack_max` 三档压力测试记录每次采样耗时、alloc bytes、peak RSS；达到内存预算时返回 degraded/partial，不允许 OOM。
- 100 个设备、10k conntrack、前端可见的建议发布门槛：Quickstart 增量 RSS ≤ 16 MiB，采样 CPU 单核平均 ≤ 10%，接口 P95 ≤ 500ms；最终数值要在最低支持硬件上基线确认。
- Inventory 内容无变化的 10 分钟内，历史文件写入次数 ≤ 1；流量轮询次数不影响该计数。
- `go test -race` 覆盖并发 Inventory/Traffic 请求；基准覆盖 20/100/500 设备和 1k/10k/65k flows。

## 7. OUI 与 Bandix 的资源边界

### 7.1 Quickstart 与 `luci-app-oui` 的 OUI 实现不同

- 参考项目 `luci-app-oui` 的 OUI JSON 约 99 KiB，`oui.js` 首次需要时加载，并在浏览器页面内常驻 `database.prefixes`；失败请求也缓存。它不会直接计入 Quickstart 后端 RSS，但会增加 Web 传输与浏览器内存。
- Quickstart 当前另有 Go 实现：`backend/service/gomanuf.go` 在进程 `init()` 中异步读取 `/usr/share/quickstart/manuf`，构造成 `map[int]interface{}` 与多组 `map[uint64]string`，初始化完成后在整个进程生命周期常驻；`GomanufSearch()` 的首次调用会等待加载结束。当前打包源文件约 1.9 MiB、55,134 行，解析后的 Go map 不能按文件大小估算。
- 因此 M12 必须分别测量文件大小、解析总分配与加载前后 Go heap/RSS，不能用参考项目的 99 KiB 浏览器 JSON 推断 Quickstart 后端成本。
- 若常驻开销超过预算，应采用生成期去重、紧凑有序前缀表或仅保留产品所需映射；禁止每个请求重复解析，也不能在没有测量前直接增加第二份 OUI 数据结构。

证据：Quickstart `backend/service/gomanuf.go`；参考项目 `luci-app-oui/htdocs/luci-static/resources/oui/oui.js` 与 `vendors-451e5311befc.json`。

### 7.2 `luci-app-bandix`

- 本 checkout 只有 LuCI 前端/RPC 代理和架构说明，没有 Rust 后端源码；“高性能”声明不能替代本项目实测。
- 随仓文档描述 1 秒实时环、1 小时长期采样和 365 天保留，还可启用连接、DNS、配额和持久化；这些都是独立内存/磁盘预算。
- README 要求 Linux 6.x、建议 OpenWrt 24.10+、主要面向单网口简单网络，并要求关闭硬件流量卸载/Turbo ACC。
- LuCI 状态页面包含 1 秒级多个轮询；直接照搬会增加请求和渲染压力。

因此接入必须满足：

- 作为 capability-gated 可选 Adapter，未安装时核心设备管理完整可用；
- 默认不启用 DNS/连接明细和长期持久化；
- 对实时点数、连接记录、DNS 记录、设备数、保留期、磁盘占用分别设硬上限；
- 安装/启用前展示内核、接口和 offload 冲突检测；
- Quickstart 只消费类型化聚合 API，不复制 Bandix 全量状态到第二套无界 map；
- 分别记录 Bandix daemon RSS/eBPF map/历史文件与 Quickstart Adapter 的资源，不把开销混在一起。

来源：`luci-app-bandix/README.zh.md`、`ARCHITECTURE.md`、`view/bandix/index.js`、`root/usr/libexec/rpcd/luci.bandix`。

## 8. 实施优先级

1. **安全阻断：** Alias/Hostname 拆分、后端严格校验、取消 raw shell interpolation、dnsmasq preflight/rollback。
2. **实时路径瘦身：** Inventory cache/single-flight、流量请求不写历史、移除 legacy 双请求。
3. **状态有界：** totals TTL/limit、批量淘汰、churn/race/benchmark 测试。
4. **conntrack 峰值：** 流式聚合、fallback 恢复、压力降级和诊断指标。
5. **长期洞察：** 在资源门槛通过后再接 Bandix 类 Adapter；历史、DNS、连接逐项启用。

这五项顺序是依赖关系：在 Hostname 安全和实时路径有界以前，不应继续叠加更多长期统计功能。
