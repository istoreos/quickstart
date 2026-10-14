# M22 五层覆盖矩阵与验收框架验收

日期：2026-09-25

## 任务树结果

- [x] 原低保真矩阵 48 项能力全部迁移为稳定 Requirement ID。
- [x] 新增 7 个 P0 需求行，每行覆盖正常、缺失、错误、恢复和重复提交场景。
- [x] 每行记录产品需求、原型、后端、正式前端和实机验收五层状态及证据或计划。
- [x] 所有尚未完成的正式 UI、Router Context、效果状态、迁移与图标持久化均保持 `partial` 或 `not_started`。
- [x] 校验器检查 ID、五层字段、状态、证据文件、完成状态与 P0 场景。
- [x] `make verify-product` 同时运行校验器自测并重建可读 Markdown 矩阵。

## 优化记录

1. 首次自检发现 Makefile 帮助列表缺少续行符；修正后重新执行统一命令。

## 证据

- 机器可读源：`docs/lan-device-management-requirements.json`
- 可读矩阵：`docs/lan-device-management-coverage-matrix.md`
- 校验器：`scripts/validate-lan-device-coverage.mjs`
- 测试结果：`docs/evidence/m22/test-results.txt`
