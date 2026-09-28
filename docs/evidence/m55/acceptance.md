# M55 设备身份与地址闭环验收

> 日期：2026-09-28；结果：通过；优化次数：0。

## 用户结果

- 设备备注名继续接受 Unicode；新增明确提示，说明 DHCP Hostname 与中文备注分开保存。
- DHCP Hostname 允许空值表示“不设置”，非空时只接受 1～63 位 ASCII 字母、数字和内部连字符；前后连字符、中文、空格、点、下划线、控制字符和超长输入均在 Store 前拒绝。
- 手工创建的 MAC 设备跨重启保持 `never_seen`，同一 MAC 首次上线后合并为一个 `online` 身份，中文备注保留。
- 同一 MAC 地址变化时旧地址进入历史；相同 IPv4 被另一 MAC 使用时不会转移身份。
- DHCP 地址池冲突计划零写入；关闭必须展示 12 台受影响设备和恢复路径；隔离 Store 中关闭及恢复均提交成功。

## 自动化与设备检查

```text
SUMMARY result=pass mode=local identity=merged addresses=stable hostname=separated dhcp=restored A=not-connected
SUMMARY result=pass mode=device B_inventory=consistent staged_dnsmasq=healthy A=read-only cleanup=armed
M55_cleanup=verified
```

- 本地聚合门禁执行 8 组关键 Go 场景及正式 UI 14/14。
- B 的真实设备清单不存在重复 Device ID，当前地址与历史地址集合不重叠。
- B 使用真实 `dnsmasq --test` 验证独立 ASCII hostname fixture，未重载服务。
- A 仅在前后读取 `network`、`dhcp`、`firewall` 哈希和 dnsmasq PID，结果一致。
- B 的临时 dnsmasq fixture 已清理。
