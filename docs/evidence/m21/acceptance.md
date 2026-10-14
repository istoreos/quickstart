# M21 唯一产品与领域基线验收

日期：2026-09-25

## 任务树结果

- [x] 将 `lan-device-management-product-architecture.md` 冻结为唯一当前产品与领域基线。
- [x] 产品分析、参考研究、能力路线图、旧里程碑和低保真均标记为历史、研究、执行计划或交互验证材料。
- [x] 唯一一级结构冻结为“设备 / 分组与计划 / 局域网设置”，规则台账归入局域网设置。
- [x] `CONTEXT.md` 删除 Device Profile 的分组归属，并补齐 Icon Preference、Desired Policy、Applied Configuration、Observed Effect、Internet Access Policy、Speed Policy、Usage Policy 和 Router Context。
- [x] 冻结设备资料、网络与上网、使用限制三个写入任务及失败语义。
- [x] 记录一次性迁移和任务级事务两项难以逆转的 ADR。

## 七个 P0

七个问题均在当前基线 1.1 节得到单一冻结结论；具体实现与五层证据由 M22～M31 继续完成。本里程碑不把“已经决定”误报成“已经实现”。

## 验收证据

- 唯一基线：`docs/lan-device-management-product-architecture.md`
- 领域词汇：`CONTEXT.md`
- ADR：`docs/adr/0001-single-product-surface-and-one-time-migration.md`、`docs/adr/0002-task-scoped-network-transactions.md`
- 执行计划：`docs/lan-device-management-m21-m31-baseline-p0-ui-milestones.md`
- 自动测试：见本目录 `test-results.txt`
