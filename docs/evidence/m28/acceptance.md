# M28 验收记录：任务级事务、幂等与恢复

日期：2026-09-25

## 任务树

- 为 Profile、Network、Restrictions 三个写入任务提供统一事务结果。
- 实现 plan/validate/snapshot/apply/verify/rollback/recover 阶段语义。
- 将幂等记录从进程内 map 改为容量 512 的持久原子日志。
- 在启动读取时把未完成事务转为 `recovery_required`。
- 为 Network 的预检、快照、写入、重载和恢复逐步注入失败。
- 为 Profile 原子写失败及 Restrictions 应用/验证/回滚失败提供结构化结果。

## 验收结论

- **通过**：相同键和相同请求跨模块重建只重放结果，不重复写入；不同请求返回冲突。
- **通过**：成功回滚为 `rolled_back`，回滚失败或进程中断为 `recovery_required`，均带恢复动作。
- **通过**：中文 Device Alias 只进入 Profile JSON，原始 ASCII DHCP Hostname 保持独立。
- **通过**：非法 Hostname、能力缺失、地址冲突均在写入前拒绝。
- **通过**：Network 保持 DHCP host、地址和网关/DNS 为同一快照与补偿边界。
- **通过**：联网权限只快照 firewall，限速只快照 eqos，任务失败不会修改另一配置域。
- **通过**：事务日志仅保存 SHA-256 请求指纹，不记录备注、Hostname、MAC/IP 请求正文。
- **通过**：正式响应已返回 task/stage/status/replayed/recoveryAction，页面无需解析内部错误字符串。

## 自检与优化

- 第 1 次：旧测试仍断言笼统 `apply_failed`；迁移为 `rolled_back` 并增加 `recovery_required` 测试。
- 第 2 次：发现事务日志保存请求原文的隐私风险；改为 SHA-256 指纹并增加泄漏回归测试。
- 未使用第 3 次优化。

## 与 M29 的边界

- M28 完成单设备 Restrictions 的 access/speed 事务 seam。
- 分组、计划、额度触发的多设备批量协调、单次服务重载和部分失败汇总由 M29 在此 seam 上完成。
