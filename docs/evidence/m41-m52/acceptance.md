# M41～M52 Quickstart NetPolicy 验收记录

> 日期：2026-09-27  
> 结论：M41～M52 通过。Quickstart 原生限速已在旁路由 B 完成真实 IPv4 数据面、事务恢复、发布回滚、资源边界和正式 UI 验收；主路由 A 全程只读。

## 1. 测试边界

| 角色 | 地址 | 本阶段权限 | 结果 |
| --- | --- | --- | --- |
| A 主路由 | `192.168.30.1` | 低频、串行、只读；禁止安装、重启、策略写入和压力测试 | 关键配置哈希前后一致，网络持续可用 |
| B 旁路由 | `192.168.30.244` | Quickstart NetPolicy 部署与可回滚策略测试 | 原生服务健康，eBPF/TCX 数据面就绪 |
| C 测试终端 | `192.168.30.93` | 经 B 访问隔离测试流量 | 2 Mbit/s 限速与回滚吞吐均实测通过 |
| D 对照终端 | `192.168.30.7` | 临时 HTTP 流量源；默认网关仍为 A | 临时路由和文件均已清理，默认链路不受影响 |

Bandix 的 UCI 与数据在 B 的 `/root/m52-backup` 保留备份。Bandix 未卸载、未改写，当前为停止且禁用；Quickstart NetPolicy 使用独立二进制、服务名、端口、配置、持久化目录、规则 owner 和内核挂载身份，不与 Bandix 共同接管数据面。

## 2. 里程碑任务树、验收标准与结果

| 里程碑 | 任务树 | 可测试验收标准 | 自检结果 |
| --- | --- | --- | --- |
| M41 独立产品骨架 | 独立仓库/分支 → 独立包与服务 → 命名空间清单 | 不复用 Bandix 二进制、服务、UCI、端口或数据目录 | 通过：`quickstart-netpolicy`、`/etc/init.d/quickstart-netpolicy`、端口 `8765` 均独立 |
| M42 领域接口 | 稳定设备身份 → 限速策略 → capability/health 契约 | API 不泄漏 eBPF map、tc 命令或 Bandix 结构 | 通过：`interfaceVersion=1`、`engine=quickstart-native` |
| M43 持久化 | 有界快照 → 原子替换 → 重启读取 → 损坏保护 | 写入不会留下半文件，重启后策略一致 | 通过：策略重启后仍为 `effective` |
| M44 事务协调 | plan → apply → verify → rollback → 幂等/版本冲突 | 失败有确定状态，可按 revision 回滚 | 通过：应用 revision 1→2，显式回滚至 1 |
| M45 数据面协调 | desired/reconciled 分离 → TCX/eBPF 加载 → 运行态核验 | 保存不等于生效；仅观察到内核状态才返回 effective | 通过：B 健康响应为 `dataPlane=ready`，规则为 `effective` |
| M46 所有权隔离 | interface owner → policy owner → 启停清理边界 | 不删除非本产品资源，不与 Bandix 同时持有测试接口 | 通过：Bandix 停止后启动原生引擎；SMOKE 无外部 owner 规则 |
| M47 资源边界 | HTTP/规则上限 → 超时 → RSS/FD 检查 | 65 KiB 请求为 413，畸形 JSON 为 400，RSS <16 MiB、FD <64 | 通过：413/400、RSS 4.7～4.8 MiB、FD 23 |
| M48 发布回滚 | 静态 PIE 构建 → 校验和 → immutable release → current/previous | 同一产物可复装，可切换 release，服务失败不假报成功 | 通过；并修复 BusyBox `mv` 跟随目标链接导致未切版的问题 |
| M49 原生事务 API | health/capabilities → policy plan/apply/verify/rollback | API 边界与领域事务一一对应，错误码稳定 | 通过：Rust 43/43，400/413 实机复验 |
| M50 Quickstart Provider | 深 Provider 接口 → native adapter → 能力与状态模型 | UI/handler 不拼装内核请求；其他功能不依赖限速引擎 | 通过：原生 Provider 注册、状态读取和事务写入通过 Go 测试 |
| M51 迁移与正式 UI | 显式扫描 → 逐项审查 → 应用/回滚 → 简洁缺失态 | 不静默迁移、不自动卸载 Bandix；空规则也可无损切换；桌面/手机可用 | 通过：空计划迁移成功，原生执行方式为默认；UI 13/13 |
| M52 四机验收 | 安全预检 → B 部署 → C/D 隔离吞吐 → 回滚/重启 → UI/业务 SMOKE | A 零写入；C 命中限速、D 不被策略污染；发布、资源、交互证据完整 | 通过：自动化拓扑、业务、限速和 36 视图 UI 全绿 |

## 3. 真实数据面证据

可重复入口：

```sh
cd /projects/workspace-linkease-ubuntu/openwrt-apps/quickstart-netpolicy
scripts/smoke/topology-smoke.sh
```

本次摘要：

```text
SUMMARY result=pass limited_Bps=251580 rollback_Bps=110695398 rss_kib=4804 fds=23 malformed=400 oversized=413 A=read-only
```

- 2 Mbit/s 目标对应约 250,000 B/s，实测 `251,580 B/s`；
- 回滚后同一路径 `110,695,398 B/s`，超过限速态 400 倍；
- 测试只添加 B→D 与 D→C 的精确 `/32` 临时路由，不改 A；退出 trap 会恢复策略并清理路由、进程和测试文件；
- 额外 10 秒资源观察中 RSS 保持约 `4,700 KiB`，FD 保持 `23`。

## 4. Quickstart 业务与 UI 证据

- Go 全量测试：`1547` 项通过；
- 正式设备 UI 静态验收：`13/13` 通过；
- 生产前端构建：通过；
- 四机业务 SMOKE：`43/43` 通过；
- 限速 SMOKE：`10/10` 通过；
- Playwright 登录态：桌面、平板、手机共 `36` 个视图，无 blocker、console error、request failure 或 interaction failure；
- 原生模式只显示“服务已就绪，请在设备详情中设置速度上限”，技术执行方式收在高级区；Bandix 迁移只在当前 Provider 为 Bandix 时出现。

## 5. 发布身份与清理状态

- B Quickstart release：`0.14.0-m52-final`；
- B Quickstart 后端 SHA-256：`e441afc8ad1f2d4dbd79947111444281a78fb48a0f083a414cf3d870014fe288`；
- B Quickstart 前端入口 SHA-256：`5b78212d10e058afe1e46bca5dd34d2ac37d5d5c563101b55d28fc187ee6c7a8`；
- B NetPolicy SHA-256：`66e517086a39232476aca0cf0e66853fd8e8bad2483d8387c45b22e0bc2a6e81`；
- B 最终 Provider：`quickstart-native`；原生服务启用且运行，Bandix 停止且禁用；
- 没有遗留 `topology-smoke` 策略、临时 `/32` 路由或 D 上的测试 HTTP 服务。

## 6. 明确未宣称完成的范围

C 当前没有可用于本验收的 IPv6 数据面，因此本记录只证明 IPv4 真实整形；IPv6 capability 和单元测试不能替代 IPv6 端到端吞吐验收。时段限速、配额跨周期、24 小时长稳继续由各自矩阵项跟踪，不因 M52 通过而自动标记为完成。
