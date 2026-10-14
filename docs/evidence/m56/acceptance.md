# M56 DHCP 分配权与能力降级闭环验收

> 日期：2026-09-28；结果：通过；优化次数：0。

## 用户结果

- 只有所选 LAN 的 DHCP 分配权明确属于本机时，上网路线才可编辑；外部 DHCP、未检测到 DHCP、证据冲突和状态读取错误均安全只读，不会自动启用本机 DHCP。
- 浮动网关与限速分别作为可选能力局部降级。任一能力缺失、停用或读取失败，不会阻塞设备浏览、联网权限、计划或额度。
- 限速能力的安装目标和启用服务统一为独立的 `quickstart-netpolicy`，不再写入 `eqos` UCI，也不再启停 Bandix/eqos 服务。
- 启用原生限速服务失败时恢复原有启动和运行状态；命令执行器封装在能力 Store 内，领域模块只处理计划、确认、状态冲突和结果。
- 安装/启用前必须先预览并再次确认；取消、失败和成功返回后均保留当前设备的限速草稿，成功不会自动提交设备策略。

## 自动化与实机检查

```text
SUMMARY result=pass mode=local authority=covered capability_degradation=isolated draft=preserved native_service=verified A=not-connected
SUMMARY result=pass mode=device A=local-editable-read-only B=downstream-read-only capabilities=available native_service=running config_drift=none
```

- 本地覆盖 DHCP authority 的 `local`、`external_observed`、`none_detected`、`ambiguous`、`error` 和错误恢复，并覆盖浮动网关/限速的缺失、错误、重试与恢复。
- 正式 UI 15/15：路线锁定不扩散到限速或联网权限；依赖准备流程使用 session draft，且不调用策略 Apply。
- 系统能力动作测试确认安装目标为 `quickstart-netpolicy`；服务启动失败时依次 `stop`、`disable` 恢复先前状态，全程不触碰 eqos。
- A 实机为 `local + editable`，仅执行读取；B 为 `downstream_router + read-only`。B 的浮动网关、原生限速和联网能力均为 `available`，原生服务运行且 Bandix 未运行。
- A、B 检查前后的 `network`、`dhcp`、`firewall` 哈希及 dnsmasq PID 完全一致，无配置漂移、无服务重启。

## 说明

B 当前静态 LAN 的运行时接口没有暴露 DHCP Server Identifier，因此实机状态保守显示 `none_detected`，而不是把默认网关臆测成 DHCP 权威；这仍正确保持只读。`external_observed` 由确定性 fixture 覆盖，符合“证据不足时不猜测”的产品基线。
