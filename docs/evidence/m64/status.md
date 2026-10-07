# M64 业务验收候选状态

日期：2026-09-28

M64 按产品决定收口“业务功能已实现、可供人工确认”的候选。连续 24 小时拓扑采样拆为独立发布门；它只验证资源趋势和拓扑长期稳定性，不再阻塞业务功能开发完成。正式发布前仍必须对最终候选重新执行完整观察，不能沿用较早候选或用加速循环替代真实时间。

## 当前候选

- Quickstart 源码：`18bde6271d319cbcd61cc85b6a791feaf5336b78`
- NetPolicy 源码：`863ed0444b63316a5d5aa6fce25be62f92d297f6`
- 发布包：`quickstart-binary-0.14.0.tar.gz`
- 发布包 SHA-256：`9685fd810ee995155b0a6b5fd263311e2e39ffdc627d47ebf515189b9e05e9ea`（连续两次干净构建一致）
- x86_64：`b0c37f06f22f4cdb8c9e8f7b47b4b4252261dee451af46c919f0947a85f7369d`
- arm64：`0c518571cd9749b70594286c9828dff1ea81129d04eff3ed238787fba1467abc`
- armv7：`38b34cb343aedb8adb1f8611d7d49f467e858765f0326008fb78bd357c496cf9`
- B 灰度回滚目录：`/root/quickstart-backups/0.14.0-18bde6271d31-20260928024700`

## 即时门禁

- 后端 1,562 项、前端 65 项、NetPolicy 43 项、部署契约、55/55 五层矩阵通过。
- B 的二进制与发布包 x86_64 哈希一致；Web、英文词典和资源版本校验通过。
- 四机拓扑、业务策略、IPv4 限速、高级工具、登录态响应式 UI 通过。
- IPv6 以 20 Mbit/s UDP 恒定负载验证 2 Mbit/s 双向 policing：下载 `2.33 Mbit/s`、上传 `2.08 Mbit/s`，回滚后双向恢复约 `20 Mbit/s`。
- 主路由 A 未部署候选；network、dhcp、firewall、dnsmasq 和 Quickstart PID 前后不变。

## 人工验收

- 面向用户任务的检查清单：[`docs/lan-device-management-manual-acceptance.md`](../../lan-device-management-manual-acceptance.md)。
- 正式确认目标为 B `192.168.30.244`；A `192.168.30.1` 只允许观察，不允许保存、应用、重启或压力测试。
- 当前范围内 39 个 P0、13 个 P1、3 个 P2 均已达到产品需求、原型、后端、正式前端和实机五层 `verified`。

## 业务完成门自检

- `make verify-lan-device-business` 通过：55 项矩阵及校验器自测、后端全量测试、65 项前端测试、TypeScript、部署契约，以及身份/地址、角色/能力、路线、事务/迁移、分组/时段、额度/历史、IPv6 和高级工具领域测试均通过。
- `make smoke-lan-device-candidate` 通过：A/B 核心 API 正常，VIP 仅由 B 持有，C 默认经 B、D 默认经 A，两端均可访问互联网和浮动网关；A 只读且未部署候选。
- B 登录态 UI smoke 通过 36 个视图：阻塞项、控制台错误、请求失败和交互失败均为 0。
- 结论：**M64 业务验收候选完成，可以进入用户人工确认；正式发布仍受独立长期观察门约束。**

## 独立长期观察

- 状态：**按产品决定延期，未计入 M64 业务完成判定**。
- 本轮短时采样于 `2026-09-28T02:49:40Z` 开始，取得 4 个健康样本后主动停止；A/B API 200，VIP 仅在 B，C 经 B、D 经 A，双方连通。
- 原始短时数据保留为 `docs/evidence/m64/soak.tsv`，但不得作为 24 小时通过证据。
- 后续应在候选冻结后单独执行 `lan-device-topology-soak.sh 86400 300`，再由分析器严格验收。
