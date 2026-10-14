# M23 信息架构与一次性迁移验收

日期：2026-09-25

## 任务树结果

- [x] 所有旧入口能力映射到设备、分组与计划、局域网设置或其中的规则台账。
- [x] 低保真明确只有一套正式目标导航。
- [x] 实现 `adopt_in_place` 迁移 Module、只读预览、版本冲突、幂等 apply、原子回执和恢复。
- [x] fixture 覆盖地址、中文旧名称、DHCP Hostname、旁路由、浮动路线、浮动网关、限速、断网与孤立规则。
- [x] 未知和冲突项进入完整报告并阻止不安全 apply，不会静默丢失。
- [x] 冻结旧页面、旧路由、旧写 Interface 和切换开关的 M30 退场清单。

## 优化记录

1. 首次实现补齐路由 fake backend 和两个迁移端点契约测试后通过。

## 说明

迁移回执只记录源版本和旧规则到新用户结果的映射，不复制策略配置，因此没有双写。OpenWrt 原生配置继续作为运行事实，由新 V2 Module 独占正式产品写入；旧写入口在 M30 删除。

## 证据

- 入口与迁移契约：`docs/lan-device-management-entry-migration.md`
- Module：`backend/service/lan_device_migration.go`
- 系统 Adapter：`backend/service/lan_device_migration_store.go`
- 测试：`backend/service/lan_device_migration_test.go`、`backend/modules/lancontrol/routes_test.go`
- 结果：`docs/evidence/m23/test-results.txt`
