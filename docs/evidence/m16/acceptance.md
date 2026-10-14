# M16 验收记录

> 日期：2026-09-24  
> 测试机：`root@192.168.9.215`  
> 回滚点：`/root/quickstart-backups/m16-20260924193358`

- 新增浮动网关状态、只读计划、事务应用和演练计划接口；产品契约只使用“优先服务节点/故障接管节点”。
- 配置经同网段、占用地址、对端、HTTP(S) URL、超时和引用校验后才能写入；typed UCI 保存并保留未知扩展项，服务失败自动恢复快照。
- 三步向导和四格状态摘要在三档视口无横向溢出；切换演练必须显式进入，当前不会自动执行。
- 测试机没有安装 floatip：API/UI 正确显示 `not_installed`，计划以 `dependency_not_installed` 零写入拒绝，未创建 `/etc/config/floatip`，没有伪造双节点健康。
- DHCP、network、firewall 哈希与 M15 完全一致；Quickstart 与 dnsmasq 正常。
- Go 全量、M16 定向 race、service 重跑 race、32 项前端测试、类型检查、生产构建、部署契约和视觉/API 验收通过。首次全包 race 命中既有并行 mock 竞争，立即独立重跑通过，非产品代码回归。

构建物：

- 后端 `7e51202af308ed356a00a5c38af5d43657f841db4f350e321299cdb539cc15f0`
- 前端 `ea8a3c18b578874985566159fb12925e3281fa86811a50557f31e8366a6f9956`

M16 结论：达成，无需功能优化轮次。
