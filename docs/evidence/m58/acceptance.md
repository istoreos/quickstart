# M58 事务与迁移验收

日期：2026-09-28

- 首次接管采用 `adopt_in_place` 收据，不复制或双写 DHCP、firewall、floatip、限速配置；预览、未知项和冲突均为零写入。
- 版本冲突返回 `conflict`；相同版本重复提交返回 unchanged，收据仅写一次。
- 快照、应用、验证、回滚均有故障注入：恢复成功返回 `rolled_back`，恢复失败返回 `recovery_required`。
- 设备资料、地址与上网、使用管理三个任务的事务测试覆盖正常、依赖缺失、步骤失败、回滚失败和重复提交。
- 正式 UI 对迁移和设备资料明确区分“原设置已恢复，可以重试”与“自动恢复未完成，请按提示处理”。
- `make test-lan-transaction-migration` 通过；12 个 DHCP/firewall/floatip/NetPolicy 隔离故障场景全部自动清理。
- A、B 两端只读请求迁移预览两次，版本稳定，network/dhcp/firewall 哈希与 dnsmasq PID 不变；未在真实路由器执行迁移 apply。

结论：`OPS-003`、`P0-MIG-001`、`P0-TXN-001` 五层证据闭环。A 保持零写入。
