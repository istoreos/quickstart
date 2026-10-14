# 局域网设备限速 P0～P2 产品与架构基线

> 状态：M41～M52 已完成；Quickstart 原生限速在 B 旁路由通过真实 IPv4 吞吐、事务恢复、发布回滚、资源边界和正式 UI 验收。IPv6 数据面仍需独立验收。  
> 日期：2026-09-27

## 1. 产品结论

用户设置的是“设备速度上限”，不是某个插件中的规则。正式界面始终围绕三件事回答：

1. 哪台设备需要限制到多少速度；
2. 应由哪个上网节点执行；
3. 当前是仅保存、已加载，还是已经验证生效。

底层执行方式只放在“局域网设置 → 设备限速服务 → 高级：限速执行方式”。普通设备详情不显示命令、配置段或 API。

## 2. 领域边界

- `DevicePolicyModule`：设备限制用例、计划、并发版本和事务结果。
- `RateLimitModule`：执行节点、稳定身份、IPv6 支持和 Provider 选择。
- `rateLimitProvider`：小接口，只暴露名称、身份能力、IPv6 能力、读取和应用。
- `nativeRateLimitProvider`：Quickstart 到独立 `quickstart-netpolicy` 事务 API 的唯一 Adapter。
- `eqosRateLimitProvider`：把固定 IPv4、UCI、服务重载和 `tc` 运行态隐藏在 Adapter 内。
- `bandixRateLimitProvider`：只负责读取既有 Bandix 规则并支持显式迁移，不作为新装默认执行引擎。
- `RateLimitSettingsModule`：全局服务配置的计划、应用、验证和回滚。

Provider 接口是测试边界。HTTP handler、Vue 页面和分组调度器不得拼装 Provider 请求或直接维护其配置。

## 3. 状态语义

| 状态 | 用户含义 | 是否可宣称生效 |
| --- | --- | --- |
| `ready` | 执行节点和稳定身份已满足，可设置 | 否 |
| `configured_not_loaded` | 规则已保存，执行服务未加载 | 否 |
| `loaded_unverified` | 服务已加载，效果尚不能确认 | 否 |
| `verified` | 配置与运行态均观察到 | 是 |
| `configured_unstable` | IP 规则存在，但设备地址会漂移 | 否 |
| `configured_wrong_node` | 规则不在设备实际上网节点 | 否 |
| `needs_address_reservation` | 轻量执行方式缺少固定 IPv4 | 否 |
| `unavailable` | 实际执行节点不在本机或状态不可读 | 否 |

流量卸载风险和 IPv6 未覆盖是独立警告，不能被“配置已保存”遮蔽。

## 4. P0～P2 交付

### P0：真实生效状态与执行节点

- 网关路线决定执行节点；旁路由、浮动网关和指定网关不会误写本机规则。
- 轻量执行方式要求当前 IPv4 与地址预留一致。
- 返回 configured、loaded、verified、IPv6 和流量卸载状态。
- 设备详情用一张紧凑状态卡展示结论与下一步。

### P1：计划化全局设置与时段限速

- 全局限速服务采用“预览影响 → 确认应用 → 验证 → 失败回滚”。
- 版本变化会拒绝旧计划；旧写入口进入相同事务模块。
- 分组和单设备计划均可选择“暂停联网”或“限制速度”，支持跨午夜。

### P2：原生 Provider 与 Bandix 迁移

- 新装默认使用独立维护的 `quickstart-netpolicy`，按 MAC 执行，不要求固定 IPv4。
- Bandix Adapter 保留为现有用户的迁移输入；迁移必须先扫描、逐条审查并由用户确认。
- 单次响应最多 1 MiB、最多 512 条规则、默认 3 秒超时；Quickstart 不复制流量历史或无界状态。
- Provider 切换前检查现有规则；支持有规则迁移和已验证的空计划切换，不会静默遗留规则。
- 分组批量写入使用同一 Provider；部分失败会补偿已经成功的写入。
- Provider 缺失、服务未运行或内核不满足时明确锁定限速并解释原因，不静默切换执行方式。

## 5. 实机验收结论与剩余隔离环境测试

主路由 `192.168.30.1` 全程只读；所有新代码和有状态验收只部署到旁路由 `192.168.30.244`，并通过以下检查：

1. B 的后端、前端入口、英文目录和浏览器缓存版本一致；
2. 四机拓扑、浮动 IP 归属、C/D 默认网关和外网可达性符合预期；
3. Provider 注册表、不可用原因、无变更 Plan、设备执行状态等 P0～P2 公共契约通过；
4. 桌面、平板和手机登录态 UI 交互通过，未发现控制台错误、请求失败或点击阻断；
5. SMOKE 前后 `network`、`dhcp`、`firewall`、`eqos`、`floatip` 五份关键配置零漂移。

真实整形已通过 C→B→D 的隔离路径完成；以下尚未完成项目只能继续在不会影响主路由承载业务的隔离环境执行：

1. IPv6：C 当前无可用 IPv6 地址，仍需补端到端数据面与实际吞吐上限测试；
2. 流量卸载：继续验证风险提示；测试不得自动关闭流量卸载；
3. 跨午夜时段：用缩短测试窗口验证进入、退出及下一次执行时间。

主路由 `192.168.30.1` 继续只允许低频、可回滚的只读检查，不执行压力测试、批量改写或为验收临时安装服务。

原生引擎的四机拓扑、真实吞吐、发布身份、资源占用和清理证据见 [`docs/evidence/m41-m52/acceptance.md`](evidence/m41-m52/acceptance.md)。Bandix 历史验收仍保留在 [`docs/evidence/p0-p2-rate-limit/bandix-runtime.md`](evidence/p0-p2-rate-limit/bandix-runtime.md)，仅作为迁移输入基线。
