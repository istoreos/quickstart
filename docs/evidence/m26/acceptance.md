# M26 验收记录：Router Context 与 DHCP 分配权

日期：2026-09-25

## 任务树

- 建立按 LAN 查询的 Router Context 领域对象。
- 归一 LAN 地址、上级网关、本机 DHCP 配置/健康与外部 DHCP 证据。
- 输出五类 DHCP authority 和后端权威的路线可编辑性。
- 失败时返回安全只读结果，不让依赖接口拖垮设备清单。
- 增加 `/v2/router-context/`、接口契约与 fixture 测试。

## 验收结论

- **通过**：`local`、`external_observed`、`none_detected`、`ambiguous`、`error` 均有确定结果。
- **通过**：只有本机 DHCP 已配置且服务健康时允许路线编辑。
- **通过**：外部 DHCP 提供主路由操作提示和可复制的本机地址，不提供自动开启 DHCP。
- **通过**：多 DHCP 证据和本机 DHCP 异常均安全只读。
- **通过**：普通主路由、旁路由、无 DHCP、两个 DHCP 证据、多默认路由、读取失败和恢复均由 fixture 覆盖。
- **通过**：接口以 `lan` 参数划定上下文，安全校验 LAN 标识；角色说明不能绕过 authority。

## 自检与优化

- 首轮实现及自检通过，未消耗优化次数。
- Swagger 契约同步了 Router Context 与 M25 的完整 Capability 状态。

## 实机边界

- 测试路由器上的 ubus、dnsmasq 运行态和外部 DHCP server-id 证据将在 M31 复验。
