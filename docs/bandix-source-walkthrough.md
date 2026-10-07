# Bandix 源码纳管与原理走读

> 基线：Bandix core/openwrt `v0.12.10`，LuCI `v0.12.11`。这些版本与 B 旁路由当前部署一致。

## 1. 为什么拆成三个仓库

Bandix 不是一个单独的 LuCI 页面，而是三个职责不同的上游项目：

| 层 | 本仓路径 | 主要职责 |
| --- | --- | --- |
| 核心 | `third_party/bandix/core/` | Rust 用户态服务、eBPF 程序、内存映射、统计、规则存储和本地 HTTP API |
| OpenWrt | `third_party/bandix/openwrt/` | 把核心二进制、UCI 默认配置和 procd 启动脚本打成 `bandix.ipk` |
| LuCI | `third_party/bandix/luci/` | Bandix 独立管理 UI，以及 `luci.bandix` rpcd 到 `127.0.0.1:8686` 的代理 |

采用 submodule 而不是复制源码，原因是：保留上游历史和许可证；Quickstart 只记录经过验证的固定提交；升级时可以清楚审查 gitlink 变化；不会把约 10 MiB 的第三方历史混入 Quickstart 主仓。

## 2. 一条数据包经历了什么

```text
LAN 设备数据包
    │
    ▼
br-lan TC ingress / egress
    │  shared_ingress / shared_egress
    ▼
Bandix eBPF
    ├─ 判断 IPv4/IPv6 是否属于本地子网
    ├─ 按源/目标 MAC 累加 LAN/WAN 字节
    ├─ 查询 MAC_RATE_LIMITS / 配额映射
    └─ token bucket 不足时返回 TC_ACT_SHOT
              │
              ▼ 每秒读取一次 maps
Rust userspace TrafficMonitor
    ├─ 邻居表/hostapd → MAC、IP、在线状态
    ├─ 累积量差值 → 实时速率
    ├─ 时间规则/默认策略 → 回写 MAC_RATE_LIMITS
    ├─ 可选内存 ring / 小时历史持久化
    └─ 127.0.0.1:8686 HTTP API
              │
        ┌─────┴──────────┐
        ▼                ▼
Quickstart Provider   luci.bandix rpcd → Bandix LuCI
```

## 3. 启动和 eBPF 加载

从以下文件开始阅读：

1. `core/bandix/src/main.rs`：只负责解析 CLI 并进入 `command::run()`。
2. `core/bandix/src/command.rs`：参数校验、读取接口子网、创建共享 eBPF、启动设备发现、监控模块和 HTTP 服务。
3. `core/bandix/build.rs`：源码构建时通过 `aya-build` 编译 `bandix-ebpf`，并把生成对象嵌入用户态二进制。
4. `core/bandix/src/ebpf/shared.rs`：加载嵌入对象，在接口上建立 `clsact`，挂载 ingress/egress classifier。内核至少 6.6 时 `auto` 选择 TCX，否则选择 netlink TC；加载失败最多重试三次。
5. `openwrt/openwrt-bandix/files/bandix.init`：把 `/etc/config/bandix` 转换为 CLI 参数并交给 procd，以 root 服务运行并配置自动拉起；当前脚本没有进一步收窄 Linux capabilities。

当前 B 的内核为 6.6.86，因此实际选择 TCX；日志已经证明两个方向均成功挂载。

## 4. eBPF 数据面

入口在 `core/bandix-ebpf/src/main.rs`。流量和 DNS 共用两个 classifier，通过 `MODULE_ENABLE_FLAGS` 决定是否执行对应模块。

限速核心在 `core/bandix-ebpf/src/modules/traffic/`：

- `maps.rs` 定义上限均为 1,024 项的 `MAC_TRAFFIC`、`MAC_RATE_LIMITS`、`RATE_BUCKETS` 和配额 maps。
- `mod.rs` 将“本地 → 外部”的 ingress 识别为上传，将“外部 → 本地”的 egress 识别为下载；IPv4 与 IPv6 使用同一 MAC 身份和同一套限速表。
- 速率单位是 bytes/s。Quickstart 的 Mbit/s 会乘以 `125000` 后写入。
- 限速算法是容量约一秒的 token bucket。令牌不足时直接返回 `TC_ACT_SHOT` 丢包，依赖 TCP 拥塞控制或应用层重试形成最终吞吐上限；它不是把包排队后平滑发送的 HTB/FQ qdisc。
- 配额耗尽也会直接丢弃 WAN 流量，但 LAN 内部流量不受影响。

因此产品上应称“最高速度”，不能承诺低抖动整形。实测时除了平均吞吐，也要观察丢包、时延和 UDP 体验。

## 5. 用户态控制面

`core/bandix/src/monitor/traffic.rs` 每秒执行一次：

1. 读取 eBPF 的累计 MAC 字节计数；
2. 用上次采样值计算增量和速率；
3. 从当前本地时间匹配定时规则；多条规则同时命中时，各方向选择最低的非零上限；
4. 合并默认限速、白名单和配额状态；
5. 把最终结果回写 `MAC_RATE_LIMITS` 与配额 maps。

这意味着时间规则的进入/退出通常有不超过一个采样周期的延迟。规则落在 `data_dir/rate_limits_schedule.txt`，API 写入后立即持久化，但真正的数据面 map 由下一次监控周期更新。

设备身份来自 `core/bandix/src/device.rs`：读取邻居表、hostapd 客户端和桥端口信息，以 MAC 聚合当前/历史 IPv4、IPv6、连接类型与在线时间。主机名还会每 600 秒从 `ubus call luci-rpc getHostHints` 刷新。

实时统计使用内存 ring；长期统计、DNS 记录和连接模块可以独立开启。我们的低资源部署关闭历史落盘、DNS 和连接模块，只保留流量/限速所需的一秒控制循环。

## 6. HTTP API 与 LuCI

`core/bandix/src/web.rs` 是一个小型 Tokio HTTP server。release 构建只监听 `127.0.0.1`，自身没有认证；安全边界依赖 loopback。`core/bandix/src/api/traffic.rs` 提供设备流量、全天计划、默认规则、白名单、配额和历史查询等接口。

`luci/luci-app-bandix/root/usr/libexec/rpcd/luci.bandix` 用 `curl` 把 ubus 调用代理到本地 HTTP API；ACL 位于 `root/usr/share/rpcd/acl.d/`，LuCI 页面位于 `htdocs/luci-static/resources/view/bandix/`。

Quickstart 不经过这层 shell rpcd，而由 `backend/service/rate_limit_bandix_provider.go` 直接访问 loopback API。这样减少一层 JSON 转换，并把超时、最大响应、最大规则数和错误语义集中在 Provider Adapter 内。

## 7. Quickstart 当前使用的能力边界

Quickstart 目前只依赖：

- `GET/POST/PUT/DELETE /api/traffic/limits/schedule`；
- MAC 身份的 IPv4/IPv6 全天设备限速；
- Provider 可用性、已配置/已加载/已验证状态。

没有直接复用 Bandix 的设备名称、完整 UI、DNS、连接列表、历史 ring、默认全网限速、白名单和配额。Quickstart 自己仍是产品领域真相，Bandix 是可替换执行 Provider，不能让其 API 数据结构渗透到页面和设备策略模型。

## 8. 走读时应重点关注的风险

| 优先级 | 观察 | 对我们的影响 |
| --- | --- | --- |
| P0 | `TrafficMonitor::apply_rate_limits()` 通过 `Arc::as_ptr` 把共享 `aya::Ebpf` 强转成可变引用 | 这是需要深入审查的 Rust aliasing/soundness 风险；升级 Aya 或增加并发访问前必须消除或证明安全 |
| P1 | `RATE_BUCKETS` 值在 eBPF 中直接读改写，未看到按 CPU 或自旋锁保护 | 多核 ingress/egress 并发可能造成桶状态竞争；要用压力测试确认误差范围 |
| P1 | 限速通过丢包而非排队 | UDP、实时音视频与高 RTT TCP 的体验可能比相同数字的 qdisc 整形更差 |
| P1 | 自制 HTTP server 单次读取 4,096 字节 | 当前 Quickstart 小请求安全，但扩大批量 API 前需修复完整 Content-Length/分段读取 |
| P1 | rpcd 脚本部分位置计算了 escaped 值却仍拼接原始变量 | 独立 Bandix LuCI 写入口需额外输入审计；Quickstart 当前不走这条写链路 |
| P2 | Bandix LuCI 的 `index.js` 很大且包含一秒轮询 | 不应整体移植到 Quickstart；只借鉴能力和数据表达 |
| P2 | OpenWrt 包的 `Build/Compile` 为空，默认封装上游 release 二进制 | 若要求供应链完全从源码可复现，需要另建 Rust/musl/eBPF 构建流水线，不能把 IPK 打包误称为核心源码编译 |

## 9. 推荐走读顺序

1. `core/bandix-ebpf/src/modules/traffic/maps.rs`
2. `core/bandix-ebpf/src/modules/traffic/mod.rs`
3. `core/bandix/src/ebpf/shared.rs`
4. `core/bandix/src/monitor/traffic.rs`
5. `core/bandix/src/storage/traffic.rs`
6. `core/bandix/src/api/traffic.rs`
7. `core/bandix/src/device.rs`
8. `openwrt/openwrt-bandix/files/bandix.init` 与 `bandix.config`
9. `luci/luci-app-bandix/root/usr/libexec/rpcd/luci.bandix`
10. Quickstart 的 `backend/service/rate_limit_bandix_provider.go`

## 10. 初始化、校验和升级

克隆后初始化：

```sh
git submodule update --init --recursive
make bandix-source-check
```

升级必须是显式评审动作：

```sh
git -C third_party/bandix/core fetch --tags
git -C third_party/bandix/core checkout <reviewed-tag-or-commit>
git add third_party/bandix/core
```

另外两个子模块同理。升级时必须同步检查 API schema、eBPF maps、OpenWrt init 参数和 LuCI rpcd 参数顺序，并重新执行单元测试、`make smoke-bandix` 及隔离环境写入 SMOKE。不要使用 `git submodule update --remote` 自动追随上游主分支。
