# M54 隔离测试实验室验收

> 日期：2026-09-28；结果：通过；优化次数：0。

## 已实现

- `lan-device-isolation-lab.sh` 默认进入本地 `self-test`，不建立 SSH 连接。
- DHCP、firewall、floatip、NetPolicy 四类 Adapter 均覆盖 apply、verify、rollback 三种失败，共 12 个确定场景。
- apply/verify 失败恢复为 `rolled_back + stable`；rollback 失败保留候选状态并返回 `recovery_required`。
- 临时目录使用显式文件清理和 `rmdir`，退出、中断与失败共用 trap。
- `remote-fixture` 必须显式设置 `LAN_LAB_MUTATION=1`，写入范围仅为 B/C/D 的独立 `/tmp/quickstart-lab-*` 目录；任何写入目标解析为 A 时以退出码 3 失败关闭。

## 自检

```text
SUMMARY result=pass mode=self-test adapters=4 fault_scenarios=12 A=not-connected cleanup=armed
isolation lab safety and cleanup ok
SUMMARY result=pass mode=remote-fixture targets=B,C,D A=read-only cleanup=verified
ops contract ok
```

远端 fixture 前后 A 的 `network`、`dhcp`、`firewall` SHA-256 一致；B/C/D 临时目录均已清理。
