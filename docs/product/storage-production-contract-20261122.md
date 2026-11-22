# 存储生产实施契约与故障验收任务

日期：2026-11-22。范围对应[目标任务树](storage-implementation-goals-20261122.md) S1–S9。
本文定义生产改造任务，不表示后端已实现。现有 RAID 接口的字段和业务能力保持兼容；
错误处理、资格检查等缺陷修复需要独立回归，不能因“业务复用”而忽略。

## 1. 最小模块与实施依赖

顺序：完整只读库存 → 共用写入准入 → 有界任务及结果 → 维护和通知 → 应用启动守卫 → 恢复点/备份。
页面消费同一个查询服务；首页是摘要投影，不能拿过滤后的首页列表作为写操作依据。
不建设任意拓扑创建器、通用 DAG 引擎或任意回滚事务。

| 任务 | 代码入口 / 最小修改 | 完成验收（生产阶段） |
| --- | --- | --- |
| B01 全量库存（S1；T6/T9） | 扩展 `backend/modules/nas/diskinventory` 原始投影；新增只读聚合，不破坏 `diskstatus` 旧响应。保留系统小分区、MD 成员、外来 PV、未知/加密签名及 holders | 系统分区和 RAID 成员均可见；未知签名不被判为空盘；新旧 UI API 契约各自通过 |
| B02 身份和准入（S1；T4） | `raid/writeflow`、`nas/disklifecycle` 的写入口共用 ValidateTarget，不能仅新 UI 使用。结构化 WWN/serial/by-id、分区/MD/LVM UUID，附容量、扇区、父关系与库存版本 | 换盘同盘符、重复/缺失身份、只读、root/overlay/swap、mountinfo、holders、Docker/Incus 占用均拒绝错误目标；先全部预检，才允许第一步清盘 |
| B03 扩充结果（S2；T1） | `raid/writeflow.Store.Grow` 及 `service/raid_write_usecase.go` 返回 error；Add 已成功后 Grow 失败必须保留阶段事实 | 注入 Add 成功/Grow 失败：返回部分完成，不伪报成功、不再次 Erase/RunAdd；前端引导读取实际状态 |
| B04 只读发现与有影响恢复（S5；T2） | 新增只读 examine/库存入口；旧 `mdadmconfig.AutoFix` 保持有影响语义。Commit/Enable/Restart 向上传错误；先构造并验证候选，避免发现失败后清空旧配置 | 空/无效 UUID、配置写入失败、重启失败均有结果；只读扫描无 UCI/组装/重启；恢复须按选定 UUID，冲突禁用；旧接口仍有明确确认 |
| B05 新建初始化（S5；T3） | 核实 `raid/writecommands.BuildCreateCommand` 统一 `--assume-clean` 的历史依据。新建使用验证过的正常初始化；仅显式恢复且满足已知条件的专项用例才考虑跳过 | 带旧随机数据成员新建 RAID1/5/6/10，读取实际初始化/同步；未完成不显示“冗余已验证”。删除/格式化不称为全盘安全擦除 |
| B06 容量来源（S2；T5） | `nas/diskstatus.BuildPartitionInfo` 区分 blockBytes、filesystemBytes、usedBytes、availableBytes；保留未知值 null，使用采样来源和时刻 | LV/MD 已扩而 EXT4 未扩时 FS 容量保持旧值；statfs 错误不输出假零；分区创建使用真实扇区范围，不拿 FS 空闲量推算 |
| B07 SMART 复用（S4；T7） | 沿用 `smart/commands`、`smart/config` 与结果接口；保留退出码、原始输出及桥接能力。按 smartctl 位标志解释结果，不能将任意非零当作正在运行 | 启动/命令失败/休眠/不支持/未知/已中止/完成独立；测试启动不等于通过；结果和计划重新打开可查；与 MD/SMART 重 I/O 互斥 |
| B08 前端读取状态（S1/S6；T8） | 正式共享查询层增加 readState、sampledAt、lastError；`pages/raid/index` 失败保留旧数据并标过期，依赖状态的写入口暂停 | API 失败、实际无盘、拔盘是三种结果；恢复查询后重新准入；轮询旧 clean 不继续显示为实时健康 |
| B09 thin 监控（S3） | 新系统 LVM provider；读取 `lvs/vgs` 结构化输出，数据/元数据/VG free 分列。检查监控服务、自动扩展和 when-full 行为；Incus VG 只由 Incus 写 | 数据满、元数据紧张、VG 无余量、监控缺失、快照持块、discard 不通各有明确状态和出口；阈值须目标设备实测，不使用演示阈值直接上线 |
| B10 任务生命周期（S2/S6） | 新小型存储任务适配器，借鉴 `app/installjob` 的锁和结果模式，但不直接把安装引擎当成 RAID 引擎 | 幂等键+请求摘要；资源及祖先锁；进程重启后核对设备事实；断网非失败；日志有界；无真实进度则仅展示阶段 |
| B11 MD 维护及换盘（S4/S5） | 读取 sysfs/mdadm 的 sync_action、sync_completed、degraded、成员角色/事件；计划与结果留在阵列 UUID 上。原盘重新加入与新盘 Recover 分开 | 检查/修复语义准确；换盘容量略小、512e/4Kn、旧事件、重复 UUID、USB、只读有资格结果；不得把 Recover 清盘语义当原盘无损恢复 |
| B12 通知（S6） | 站内持久事件 + 一种服务端可配置投递适配器，沿用项目鉴权/配置；Webhook 防泄密和目的地址策略单独核实 | 按资源UUID+类型合并；已读不消除告警；恢复写入恢复时间；发送失败可重试/可查；页面关闭也投递；配置测试不导出密钥 |
| B13 开机守卫（S3） | OpenWrt procd/block-mount/热插拔协调：md → LVM → 指定 FS UUID 挂载 → 应用。核对 mountinfo 对象/读写/容量，不以目录存在或 sleep 1 判定成功 | 正常/缺盘/延迟设备/UUID 错/只读/overlay 目录占位均测试；失败保持应用停止，不写空挂载点。Docker 和 LightHost 分别验证重启与恢复 |
| B14 应用接入（S3） | Docker 当前 data-root 和新目标独立；停服务→复制→校验→切换→启动验证；旧数据保留。LightHost 管理文件交接由其模块接收稳定 ID | 复制前/复制中/切换后/启动失败均能按阶段处理；两处新写入时不能自动回退；沙箱 LVM 等启动恢复验证后开放 |
| B15 恢复点与备份（S7） | 所有者提供一致性快照和只读视图；工作台只聚合归属、覆盖、保留和验证状态；backuprepo 消费租约视图 | freeze 超时 finally 解冻；跨卷一致性策略、外部 bind mount 排除；VM 来宾文件系统识别；释放/到期/进程崩溃回收；独立位置恢复验证 |
| B16 别名与诊断（S8） | 别名按稳定资源 ID 保存；诊断仅白名单版本/拓扑/采样/阶段/脱敏日志，排除用户文件/账号/凭据/命令环境 | 换盘符别名仍对应同设备；诊断字段白名单检查；支持资料大小有界，不依赖故障目标盘读取 |

## 2. 新查询与计划协议（建议，不是已存在 API）

### 统一库存

```ts
type ReadState = 'fresh' | 'stale' | 'unavailable'; // 无设备是 fresh + resources: []
type StorageResource = {
  id: string; kind: string; parents: string[]; owner: string;
  identity: { wwn?: string; serial?: string; uuid?: string; reliable: boolean };
  currentPath: string; sampledAt: string; revision: string;
  capacity: { blockBytes?: number; filesystemBytes?: number;
    usedBytes?: number; availableBytes?: number; metadataUsedPercent?: number;
    vgFreeBytes?: number; source: string };
  dependencies: { id: string; purpose: string }[];
  capabilities: { operation: string; allowed: boolean; reasons: string[] }[];
};
```

内部容量计算使用整数扇区/字节；超出 JSON 安全整数范围时用十进制字符串。
分区范围必须有真实起止扇区、对齐与分区表类型。不能使用图形显示比例或四舍五入 GiB 作为写参数。
接口缺字段保留未知，不推断零；可写能力由后端当前事实提供。

### 固定步骤任务

1. `preflight(operation, resourceIds, revision, parameters)` 返回 planId、到期时间、影响范围、保留范围、稳定身份、步骤和禁止原因。
2. `apply(planId, confirmationDigest, idempotencyKey)`：锁定资源及祖先，在锁内重新读取和验证；变化则计划失效，不能静默改目标。
3. 同幂等键同请求返回原 taskId；同键不同请求拒绝。多客户端相同资源冲突返回当前任务入口。
4. 任务阶段：等待/执行中/成功/失败/未执行。另记录查询状态；失败阶段包含命令退出码、脱敏输出、已产生资源和下一步。
5. 磁盘写入后、记录成功前进程崩溃：重启核对目标资源和步骤事实，标记需人工检查，不能靠“未记录成功”重新执行破坏步骤。
6. 内核 MD reshape/recovery 生命周期独立于提交成员变更的 HTTP 请求；进入同步不等于全部完成。

取消只对尚未开始且支持取消的步骤开放；执行中的擦盘/reshape 不提供任意取消和自动逆向回滚。
RAID 旧即时接口暂不强行迁入新分区计划协议；服务端共享准入和阶段记录即可，逐步兼容 taskId。
JBOD 前端 `jbod` 与后端 `linear` 的已知契约差异也需独立决议和回归，不能由 mock 偷改。

## 3. 持久化与重启

任务与事件存放在启动可用、受保护且容量有界的位置，不依赖正在卸载/重建的目标文件系统。
先持久化 intent，再执行；状态更新用原子写入和落盘策略，保持最后已知结果。
限制条数、单步输出大小、保留时长与写入频率；超限仍保留未完成任务和必要恢复信息，避免写满 overlay。
预检/执行锁与所有旧写入口共用；单机进程内 mutex 不足以防其他服务或命令并发，需服务端约束与运维边界。
服务重启后将执行中任务转为“核对中”，重建实际资源关联；浏览器不能根据超时自判失败。

## 4. 平台能力与上线门槛

目标组合逐项登记：iStoreOS、内核、mdadm、LVM/device-mapper/thin 工具、EXT4 扩展工具、
SMART/USB 桥接、procd/block-mount、Incus；能力未验证则禁用对应写入，只保留只读信息。
不照搬 ZFS scrub、自修复、厂商全平台硬盘黑名单或 systemd 依赖。
RAID5/6 reshape 的备份文件及掉电恢复材料按目标工具/布局确认，保存在不受该变更影响的位置。
LVM 配置备份不是 thin 元数据备份，也不是用户文件备份。

| 故障验收组 | 必须验证的案例 | 对应任务 |
| --- | --- | --- |
| H1 身份/占用 | 同路径换盘、缺/重序列号、UUID 冲突、root/overlay/swap/holders、应用占用、状态过期 | B01/B02/B08 |
| H2 单步失败 | Add 成功/Grow 失败；配置 Commit/Restart 失败；mkfs 失败；挂载失败；FS 未扩展 | B03/B04/B06/B10 |
| H3 并发/重启 | 双标签重复提交；祖先资源冲突；写入后落记录前崩溃；服务/整机重启查询恢复 | B02/B10/B13 |
| H4 MD 生命周期 | 初始同步；拔盘/原盘重现/新盘略小；重建中断/完成；只读扫描冲突；reshape 恢复 | B04/B05/B11 |
| H5 thin 生命周期 | 数据满/元数据满；无 VG 余量；监控缺失；快照持块；TRIM 不通；扩展 FS 失败 | B06/B09 |
| H6 应用/备份 | 缺盘启动不落 overlay；Docker 各迁移阶段失败；外部目录覆盖；租约失效/崩溃回收；独立恢复 | B13/B14/B15 |
| H7 维护/告警 | SMART 命令非零/休眠/不支持/中止；维护冲突；通知失败/去重/恢复/离线投递 | B07/B11/B12 |
| H8 前端/兼容 | 现有 RAID 七类请求不变、普通 EXT4 保留、390/768/1600/暗色/键盘、国际化、诊断脱敏 | B06/B08/B16 |

先用可丢弃 QEMU 磁盘和故障注入跑 H1–H8，再在 `root@192.168.30.7` 核对当前版本、
设备身份及可销毁磁盘范围，独立执行目标机验收。本轮没有远程格式化、安装或重启。
仅 H 全部符合对应能力门槛后，才将相应写操作从原型提升为生产功能。

## 补充：系统能力不是设备资格

能力状态、操作依赖、过期处理和后端执行前复核见 [存储能力与操作资格](storage-capability-interaction.md)。能力探测与资源预检必须同时满足，前端置灰不能代替后端鉴权；当前能力交互只在产品原型实现，生产接口尚未实现。
