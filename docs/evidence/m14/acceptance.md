# M14 验收记录：Gateway Profile 后端基础

> 日期：2026-09-24；测试机：`root@192.168.9.215`

## 任务树完成情况

- [x] 建立 Gateway Target、只读 Plan、事务 Apply 和 References 接口。
- [x] 将本机、上级路由、自定义/旁路由和浮动网关统一为稳定 Target；外部契约不暴露 DHCP tag。
- [x] 兼容既有 tag、option 3/6 和 host 引用；未知 DHCP option 以 unsupported 保留。
- [x] 加入同网段、地址冲突、引用、版本和续租状态预检。
- [x] 写入使用 typed UCI、dnsmasq 校验、快照和失败恢复；重复提交不重复写入。

## 验收结果

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| 既有 target 无损映射且内部 tag 不泄露 | 通过 | `TestGatewayPolicyListsOpaqueTargetsAndPreservesUnsupportedOptions` |
| Plan 零写入、Apply 幂等、版本冲突拒绝 | 通过 | `TestGatewayPolicyPlanIsReadOnlyAndApplyIsIdempotent`、stale version 测试 |
| 被引用 target 禁止直接删除 | 通过 | `TestGatewayPolicyRejectsReferencedDeleteAndStaleVersion` |
| 2048 引用扫描低于 500 ms | 通过 | `TestGatewayPolicyRejectsOutsideLANAndScans2048References` |
| UCI 字段保留与失败回滚 | 通过 | `TestMutateGatewayPolicyPreservesHostAndMaterializesTarget`、rollback 测试 |
| 真机只读和无变化提交 | 通过 | `api-check.cjs`：Plan 可执行、changes=0、Apply changed=false、配置哈希不变 |

M14 达成；后续 M15 在此契约之上提供用户可理解的“上网路线”。
