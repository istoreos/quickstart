# M29 分组批量执行与低资源验收

状态：通过（仓库与约束内测试）；实机重载计数留待 M31。

## 任务结果

- 策略优先级固定为系统默认、全局策略、低到高分组优先级、单设备例外；来源链按相同顺序输出。
- 调度先计算全部 Effective Policy 与签名差异；未变化设备不写入。
- 默认执行器先对整个批次做能力和路线预检。预检失败时配置零写入，期望规则保留并显示降级。
- access、speed、route 写入先收集，再分别对 firewall、eqos、dhcp 各提交一次；相关服务每批最多 reload 一次。
- reload 失败时使用批次前快照恢复；恢复失败逐设备返回 `recovery_required`。
- 批次结果包含 `all_success / partial / failed / unchanged`、逐设备状态、失败原因、来源链、写入计划及 reload 次数。
- 单个模块仅有一个分钟级 scheduler；没有每设备 goroutine 或 ticker。事件 256、成员 2048、设备例外 512，批次与去重状态随有效成员回收。

## 可重复验证

```sh
cd backend
go test ./service -run 'Test(CollectedGroupBatch|DefaultGroupBatch|DeviceGroup)' -count=1
GOMEMLIMIT=192MiB go test ./service -run 'TestDeviceGroup(BatchPerformanceBudgets|HundredThousandMemberChurnRemainsBounded)' -count=1 -v
go test -race ./service
```

约束测试结果：20、256、2048 台均在 64 MiB 单次分配预算和 3 秒预算内；2048 台用例约 0.03 秒。100,000 次成员 churn 约 1.28 秒，去重 map、批次和事件均保持上限。测试设置 `GOMEMLIMIT=192MiB`，比 256 MiB 设备预算更严格。

## 边界

- 仓库测试证明提交与 reload 调用次数；OpenWrt 真实 UCI 文件、init 脚本调用和 256/512 MiB RSS 由 M31 复验。
- 额度统计沿用轻量本地采样与五分钟写盘门；正式分组编辑中的额度预览由 M30 完成。
