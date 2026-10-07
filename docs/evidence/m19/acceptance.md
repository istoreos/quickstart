# M19 验收记录：历史流量、流量配额与可选 Adapter

> 日期：2026-09-24；测试机：`root@192.168.9.215`；备份：`/root/quickstart-backups/m19-20260924122500`

## 任务树完成情况

- [x] 独立 `TrafficInsights` 模块，不扩大实时采样模块状态。
- [x] 小时/日/月分层、固定保留期、5 分钟写节流与 2 MiB 硬预算。
- [x] 日/周/月额度及 notify/block；阻断会记录并在新周期恢复先前联网状态。
- [x] 存在启用额度时启动唯一一分钟采集循环；无额度时不构成后台消费者，实时采样仍会自动停止。
- [x] 额度阻断被其他策略临时覆盖时会重新施加，修改额度/动作或禁用额度会立即恢复原联网状态。
- [x] 本地轻量 Adapter 默认可用；Bandix 仅探测，不修改 eBPF、网口或硬件卸载。
- [x] 历史用量只进入设备详情，列表仍维持实时摘要六列。

## 验收结果

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| 无 Bandix 安全降级 | 通过 | 真机返回 `bandix_not_installed`，local=`available` |
| 今天/本周/本月、重启保留 | 通过 | API quota create → restart → verify；趋势存储单测 |
| 固定预算、分层淘汰、写节流 | 通过 | 2,097,152 B 上限；`TestTrafficInsightsPersistsTrendsAndThrottlesWrites` |
| 周期重置、时区、计数器回绕、身份隔离 | 通过 | `TestTrafficQuota*`、`TestTrafficInsightsCounterResetAndIdentityIsolation` |
| 关闭页面后的额度可靠性 | 通过 | `TestTrafficInsightsBackgroundCollectorRunsOnlyForEnabledQuota`；仅有启用额度时每分钟采样 |
| 规则冲突后的额度重申与立即恢复 | 通过 | `TestTrafficQuotaReassertsBlockAfterAnotherPolicyAllowsAccess`、`TestTrafficQuotaEditRestoresBlockImmediately` |
| 长期身份状态上限 | 通过 | baseline/seen 最多 2048；`TestTrafficInsightsBaselineIdentityChurnIsBounded` |
| 20 台×30 天 | 通过 | `TestTrafficInsightsTwentyDevicesThirtyDaysStayWithinBudget` |
| 前端信息密度与语义 | 通过 | 详情按需展开，明确“不是速度上限或定时限速”；39 个前端测试 |
| 全量、race、构建、两视口 | 通过 | Go `./...`、定向 `-race`、Vite build、桌面/移动截图 |

## 真机结果与恢复

- 后端 SHA-256：`cb7da998c0f9a1bde50133f03caea7f1ec02d877d1366568e5c48d7eee3ef0bd`。
- Web 缓存版本：`0.14.0-r12`。
- 测试配额已禁用；测试前历史文件不存在，验收后已恢复为不存在。
- DHCP/network/firewall 哈希仍为 `e158cc…` / `86fdfc…` / `7f7019…`。
- 视觉脚本首轮只看到“在线”筛选空集，调整为“全部设备”后通过；产品代码无需修改。
