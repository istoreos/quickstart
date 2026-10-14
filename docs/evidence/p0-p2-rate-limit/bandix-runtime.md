# Bandix 下载、构建、部署与 eBPF 实机证据

日期：2026-09-26

## 结论

Bandix 已安装并常驻运行在旁路由 B（`192.168.30.244`），Quickstart 已选择 Bandix Provider。真实 eBPF 程序在 `br-lan` 的 ingress 与 egress 均成功挂载；Quickstart 的“预览 → 应用 → 验证 → 删除”规则闭环已在主机 C 完成，最终规则表为空。

主网关 A（`192.168.30.1`）只接受只读检查，本轮没有安装 Bandix、没有重启服务、没有修改网络配置。

## 源码与可复现构建输入

| 项目 | 位置 | 版本/提交 |
| --- | --- | --- |
| LuCI 前端 | `third_party/bandix/luci` | `v0.12.11` / `0bd9b61740c8798ca86317eb6e0784fb32dd51f5` |
| OpenWrt 包定义 | `third_party/bandix/openwrt` | `v0.12.10` / `0cc5963fe009575fe20dfd1aa2dcf9b9f05f6712` |
| Bandix Rust 源码 | `third_party/bandix/core` | `v0.12.10` / `b28c8b52d973cbf6e607eec26f15c032c75b56cc` |
| OpenWrt SDK | `/projects/workspace-linkease-ubuntu/bin/openwrt-sdk-cache/openwrt-sdk-24.10.1-x86-64.tar.zst` | SHA-256 `ff013b90ea4912c2c9fd55aef6b399fedf2bb3e1d702a3e62ef17afcafdab221` |

使用官方 OpenWrt 24.10.1 x86/64 SDK 编译了 OpenWrt 包。上游 `openwrt-bandix` 的 `Build/Compile` 明确为空：核心包按上游设计校验并封装 Bandix v0.12.10 官方静态二进制；LuCI 与 OpenWrt 包结构则由本地 SDK 生成。Rust 完整源码已作为固定版本 submodule 纳管，未冒充为本地 Rust 重编译产物。

构建产物位于项目的 `bin/bandix/`：

| 文件 | SHA-256 |
| --- | --- |
| `bandix_0.12.10-r1_x86_64.ipk` | `b388d0d14ec2810dd76fedf7607141d01e83e01cf5ac76d9dcd7dc322f227996` |
| `luci-app-bandix_0.12.11-r1_all.ipk` | `13b972eba13aaa9a8e3cc157de7402f0e7ae3e6aa49ff5d44da059f4f1240a81` |
| `luci-i18n-bandix-zh-cn_26.249.50374~1588729_all.ipk` | `7795372063e93310341e7a3c730d495dd8d60be53dac30ab970d52827ba94a73` |

## eBPF 与资源证据

| 检查 | A 主网关 | B 旁路由 |
| --- | --- | --- |
| 系统 / 内核 | iStoreOS 24.10.2 / 6.6.93 | iStoreOS 24.10.1 / 6.6.86 |
| 架构 | x86_64 | x86_64 |
| BTF | `/sys/kernel/btf/vmlinux`，4,959,890 字节 | `/sys/kernel/btf/vmlinux`，4,921,497 字节 |
| bpffs | 已挂载 | 已挂载 |
| `kmod-sched-bpf` | `6.6.93-r1` | `6.6.86-r1` |
| 真实 eBPF 加载 | 未执行 | ingress、egress 均成功 |

B 的 Bandix 配置只启用实时流量模块：历史落盘、连接展示、DNS 记录均关闭。最终进程资源为 `5,120 KiB RSS / 3 threads / 22 FD`；自动门禁上限为 64 MiB RSS。

仅检查文件和模块不足以证明 eBPF 可运行，因此又执行了真实加载：日志明确记录 shared ingress/egress program attached，流量 API 返回局域网设备数据。这构成 B 的运行时支持证明。

## 集成缺陷与修复

实机联调发现并修复三个仅靠模拟测试不容易暴露的问题：

1. Bandix v0.12.10 返回 `data.limits` 嵌套结构，Adapter 原先只接受数组或 `data` 数组；现已兼容真实响应并保留响应大小、规则数和超时上限。
2. Bandix 的全天计划是闭区间 `00:00–23:59`；`24:00` 会返回 500。Adapter 与测试现统一使用 `23:59`。
3. 策略版本原先包含每次读取都会变化的 `observedAt`，导致预览后应用必然冲突；版本现只覆盖设备身份和可配置策略，排除实时观测字段。Bandix 计划也不再误报会重载 eQoS。

另外，全局切换第一次验证失败时事务已自动回滚；修复后重新计划并提交成功。Bandix 不使用总带宽，后端现在保留现有 eQoS 总带宽，避免隐藏字段制造不可验证的伪变化。

## SMOKE 结果

- `go test ./...`：1,533 项通过（89 个包）。
- P0～P2 只读专项：`10/10` 通过，A/B 关键配置零漂移。
- 四机业务公共契约：`43/43` 通过。
- `BANDIX_MUTATION_SMOKE=1 ./scripts/ops/bandix-runtime-smoke.sh`：`19/19` 通过。
- 正式 Quickstart 链路：C 的 1000/1000 Mbit/s 高上限测试规则创建成功，状态为 `verified`；C 经 B 的默认路由、外网和 DNS 正常；随后通过 Quickstart 删除，状态恢复为 `ready`。
- 最终 Bandix 规则 API：`limits: []`。

可重复执行：

```sh
make smoke-bandix
BANDIX_MUTATION_SMOKE=1 make smoke-bandix
```

默认只读；写入模式使用未占用的 locally-administered MAC，退出时带兜底清理，并拒绝在 `192.168.30.1` 上执行写入。

## 当前部署与回滚

- B 已安装：`bandix 0.12.10-r1`、`luci-app-bandix 0.12.11-r1`、中文目录包。
- B Quickstart 二进制 SHA-256：`78021851a00f022fdfe0237324a3af61b0b98fdc4c6f34d3daa364bf913786b1`；资源版本：`0.14.0-bandix-p0p2-fix4`。
- Bandix 启用前备份：`/root/quickstart-backups/bandix-enable-20260926113714`。
- 最终 Quickstart 回滚目录：`/root/quickstart-backups/0.14.0-b60f6e53dd6a-20260926114401`。

尚未宣称完成：C 当前没有可用于验收的 IPv6 地址，因此 IPv6 真实数据面限速仍待独立测试；也没有在承载业务的主网关进行吞吐压测或安装 Bandix。
