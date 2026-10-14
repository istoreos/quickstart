# M57 上网路线闭环验收

> 日期：2026-09-28；结果：通过；优化次数：0。

## 用户结果

- 路线优先级固定为局域网默认（`default` 跟随）→ 全局默认 → 按优先级排序的设备分组 → 单设备例外；结果同时返回来源链，不依赖设备或配置遍历顺序。
- 每个非默认路线在同一 dnsmasq tag 中生成相同目标的网关 option 3 与 DNS option 6；用户界面继续只显示“上网路线”，不暴露 DHCP 实现字段。
- 只有内核邻居表明确标记为 `FAILED` 才判定“网关不可达”；无探测证据不会被误报。不可达目标在计划阶段拒绝，保持零写入。
- 地址占用、网关不在当前 LAN、网关不可达现在使用独立错误码和可执行文案，不再统一显示成含糊的“请检查填写内容”。
- 规则台账为缺失、不支持或不可达路线展示下一步，并提供“恢复默认路线”动作；动作仍需 Plan、确认、Apply，失败按原快照恢复。

## 自动化与实机检查

```text
SUMMARY result=pass mode=local precedence=deterministic gateway_dns=paired unreachable=actionable rollback=verified A=not-connected
SUMMARY result=pass mode=device C_path=B D_path=A floating_target=available gateway_dns=paired A=read-only B=read-only config_drift=none
```

- 本地测试覆盖默认/分组/设备优先级与来源链、option 3/6 同目标、显式 FAILED 邻居、地址冲突、联合地址与路线事务、dnsmasq 重载失败回滚，以及规则修复不删除地址预留。
- 正式 UI 16/16；桌面表格和移动卡片均呈现问题原因、下一步和恢复入口，中文与英文目录均完整。
- A 的真实 Gateway Target 清单包含浮动网关 `192.168.30.3`，所有带 DNS 的路线均满足唯一 DNS 等于网关地址。
- C 的真实默认路由为 B（`192.168.30.244`），D 的真实默认路由为 A（`192.168.30.1`），两端均可到达各自网关。
- 因 A 是关键网关，本里程碑未对 A 执行路线写入或续租；优先级变更和失败恢复由隔离 Store/配置 fixture 验证。A、B 前后配置哈希与 dnsmasq PID一致。
