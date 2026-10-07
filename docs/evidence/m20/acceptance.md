# M20 验收记录：高级诊断、能力真值、审计与策略备份

> 日期：2026-09-24；测试机：`root@192.168.9.215`  
> 回滚点：`/root/quickstart-backups/m20-20260924124500`

## 任务树完成情况

- [x] 明确 IPv6 预留、连接/DNS 洞察、管理探测、网络隔离、多 WAN 和识别更新的真实能力状态。
- [x] 管理探测只允许 Inventory 当前私有/链路本地地址及 80/443/8080/8443，2 秒超时、并发上限 4、禁止重定向。
- [x] 审计记录新设备、上下线、地址冲突、策略变更、真实持有者变化和配额超限；地址冲突不记录 IP，DNS reason 强制留空，事件环上限 512、设备状态上限 2048。
- [x] Webhook 默认关闭，事件白名单、队列 64、最多 5 次重试、5 秒超时，URL 回读脱敏且不跟随重定向。
- [x] 设备分组、设备例外与流量配额支持版本化导出、4 MiB 单 JSON 上限、校验和、dry-run、计划校验和分组版本校验、明确确认、事务应用和失败回滚；导入成功后立即重算有效分组策略，相同内容重复导入零写入。
- [x] 高级能力只在设备详情按需展开，默认六列和移动卡片不增加信息负担。

## 验收结果

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| 能力真值、不伪造缺失能力 | 通过 | 当前固件 IPv6 reservation=`unavailable`，Bandix=`not_installed`，segmentation/multi-WAN=`unavailable` |
| LAN-only 探测与 SSRF 边界 | 通过 | 公网地址、非清单地址、非白名单端口/协议拒绝；并发上限测试通过 |
| 审计隐私、上限与重启生命周期 | 通过 | `TestNetworkAudit*`；新设备/离线/在线跨重启顺序正确 |
| 地址冲突与真实网关切换事件 | 通过 | `TestStaticAddressConflictIsAuditedWithoutLeakingAddress`、`TestNetworkAuditRecordsObservedFloatingGatewayHolderChanges` |
| Webhook 安全默认值 | 通过 | 默认关闭、凭据 URL 拒绝、回读只显示 `configured`、重定向拒绝 |
| 导入 dry-run、确认和回滚 | 通过 | 不兼容版本/未确认拒绝；注入流量存储失败后分组恢复 |
| 可选洞察能力真值 | 通过 | 仅检测到 Bandix 不会宣称连接洞察可用；DNS 默认不采集、不记录 |
| 真机 API 与重启持久化 | 通过 | `api-check.cjs create → service restart → verify` |
| 三档视口渐进披露 | 通过 | `desktop-1440x900.png`、`compact-1024x768.png`、`mobile-390x844.png`；无横向溢出 |

## 自动门禁和真机恢复

- Go 全量测试、`go test -race ./service ./modules/lancontrol`：通过。
- 前端 42 项测试、`vue-tsc --noEmit`、Vite 生产构建：通过。
- OpenAPI YAML 已解析；`git diff --check`：通过。
- 最新后端 SHA-256：`2b8dc4db6b8f902e3b3ac2098944c1390dcf02f784c42c6b1ef88f1c4cfcf686`；前端缓存版本 `0.14.0-r8`，远端 `index.js` / `style.css` 与本地 dist 哈希一致。
- API 验收临时创建的 audit/groups/traffic 文件已删除并重启服务；最终 DHCP/network/firewall 哈希分别保持 `e158cc…`、`86fdfc…`、`7f7019…`。
- 最终真机 API 使用临时 `m20_acceptance` 分组和高额度配额执行 create → restart → verify，确认两类数据跨服务重启持久化；清理后未再调用会重建审计文件的 Inventory API。
- 视觉脚本首轮文本定位受页面翻译影响；优化 1 改为稳定组件选择器后通过。产品实现无需视觉返工。

M20 结论：达成。未支持项通过能力真值诚实降级，不提供无效开关。
