# M27 验收记录：上网路线的期望、应用与观察效果

日期：2026-09-25

## 任务树

- 用 Desired Policy、Applied Configuration、Observed Effect 替换单一 `effectState`。
- 记录有界、可跨服务重启恢复的策略效果状态。
- 通过 dnsmasq lease 文件观察保存后的新租约，但不声称终端网关/DNS 已采用。
- 统一生成同一路线 IP 的 DHCP option 3 与 option 6。
- 将等待、不可验证、观察错误和应用失败映射为“需要处理”原因。
- 更新后端模型、正式前端当前消费点、Swagger 和五层矩阵。

## 验收结论

- **通过**：任何代码路径均不再返回 `active/已生效` 作为终端效果。
- **通过**：写入成功首先为 `server_applied + pending_renewal`。
- **通过**：新租约仅升级为 `lease_observed_unverifiable`，明确终端实际网关和 DNS 仍不可验证。
- **通过**：租约源缺失、读取失败和恢复均有确定状态；读取失败不阻塞策略本身。
- **通过**：效果记录上限 2048，并以原子 JSON 持久化；服务重建后可恢复终态。
- **通过**：已有仅 option 3 的路线在再次分配时会同一事务补齐 option 6，二者使用同一 IP。

## 自检与优化

- 第 1 次检查发现旧测试/前端仍依赖 `effectState`；已整体迁移到三段模型，并使用可控租约 observer 消除环境偶然性。
- 第 1 次优化后全部回归通过；未使用第 2、3 次优化。

## 实机边界

- 真实客户端续租前后、dnsmasq lease mtime 与 option 3/6 下发将在 M31 验证。
