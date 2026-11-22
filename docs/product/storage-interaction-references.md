# 硬盘与存储：交互参考记录

记录日期：2026-11-21。本文记录交互依据及本项目的取舍，不代表实现了参考产品的全部能力。
配套设计见[日常场景与实现边界](storage-scenarios-and-implementation.md)，可运行原型见
[预览说明](ui-design/README.md)。
后续官方产品对比见[运维与故障恢复调研](storage-third-party-research-20261122.md)，
具体交互缺口、本地代码依据和实施顺序见[差距评审](storage-gap-analysis-20261122.md)。

## 参考产品与采用方式

| 产品与官方来源 | 参考的交互 | quickstart 的采用方式 | 不直接沿用的部分 |
| --- | --- | --- | --- |
| [Windows 磁盘管理概览](https://learn.microsoft.com/en-us/windows-server/storage/disk-management/overview-of-disk-management) | 多盘管理的整体视角；区分分区、未分配区域和受保护系统区 | 首页摘要进入多盘布局；点选区域查看详情和可用操作；系统区标记保护 | 不把 Windows 的盘符、动态磁盘或支持的调整操作映射成 Linux 能力承诺；Windows 的多盘聚合另有 Storage Spaces |
| [DiskGenius 创建分区手册](https://www.diskgenius.com/manual/create-new-partition.php)、[官方界面用户手册](https://www.diskgenius.com/manual/DiskGenius_User_Guide.pdf) | 分区图；新分区先存在于内存，保存后才写入分区表并格式化 | 「加入待执行 → 检查影响 → 应用 → 查看逐项结果」；应用前可撤销 | 不继承无损拆分、分区恢复、任意移动；不声称已执行操作可整体撤销 |
| [Synology Storage Manager 总览](https://kb.synology.com/en-uk/DSM/help/DSM/StorageManager/overview)、[官方存储管理规格](https://www.synology.com/en-global/dsm/7.4/software_spec/storage_management) | 集中呈现健康、硬盘、存储池与卷；显示成员和维护状态 | 首页优先提示异常；从池追溯物理来源，从硬盘追溯用途；成员加入和重建完成分别展示 | 不移植 SHR、SSD 缓存、自动修复等能力；不强迫普通 EXT4 文件区经过创建池和卷 |
| [RHEL / Cockpit 存储管理](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/7/html/managing_systems_using_the_rhel_7_web_console/managing-storage-devices-in-the-web-console_system-management-using-the-rhel-7-web-console) | Linux 物理设备、RAID 和 LVM 层次的统一展示 | 在详情中显示来源与管理归属，技术信息折叠 | 默认界面不要求用户依次创建 PV、VG、thin-pool；通过用途向导生成受支持布局 |

表格中的「采用方式」是本项目的设计判断，不是参考厂商对 quickstart 的背书。
Windows 官方概览用于确认磁盘任务与系统分区保护；多盘条形图的视觉参考也记录在 DiskGenius 官方用户手册中。
Synology 总览页使用动态加载，自动读取可能只返回页面外壳；其官方规格页作为池、卷、硬盘和维护能力的补充依据。
此处保留链接和简要说明，不复制第三方手册或整页截图。参考的是交互模式，不是照搬视觉皮肤。

## 与本项目技术约束的衔接

1. **普通文件存储保持独立路径。** 已有 EXT4 的首选是「挂载并使用」，新空间默认准备普通文件存储。
   不把“文件系统内部剩余空间”画成“尚未分区的空间”。
2. **分区 / LVM 的布局与容量变更先规划。** 格式化、删除分区、建池等进入统一计划；
   挂载、卸载、进入应用管理采用单项操作反馈，避免所有动作都绕一次计划列表。
   RAID 按 2026-11-22 的[复用约定](raid-existing-business-integration.md)保留已有业务与接口；确认改用当前主题下的页面内核对。
3. **可视化统一，写入责任明确。** [Incus LVM 官方文档](https://linuxcontainers.org/incus/docs/main/reference/storage_lvm/)
   明确其对 VG 的管理假设。因此 LightHost / Incus 的 VG 与系统应用 VG 分开；通用页面不能直接操作其内部卷。
4. **健康提示需要可处理的下一步。** 「空间不足」「成员失联」必须定位到具体对象和处理入口；
   没有经过验证的处理能力，应明确说明条件和替代路径。
5. **保留少量主操作。** 面向用户显示“文件、主机、应用、镜像保护”等用途；
   VG、thin-pool、元数据等信息放在需要判断容量或排障的位置。

## 当前原型中的落点

| 交互决定 | 原型位置 | 可检查的材料 |
| --- | --- | --- |
| 多盘布局和区域详情 | `web/src/design/StorageDiskMapPrototype.vue`、`StorageWorkbenchPrototype.vue` | [总览截图](ui-design/storage-workbench/01-overview.png) |
| 规划、撤销、应用和逐项结果 | `storageWorkbenchPrototype.ts`、`StorageWorkbenchPrototype.vue` | [待执行截图](ui-design/storage-workbench/02-pending.png)、[失败截图](ui-design/storage-workbench/07-failed-plan.png) |
| 池、卷、容量与归属 | `StorageWorkbenchPrototype.vue` | [池与卷截图](ui-design/storage-workbench/03-pools.png) |
| RAID 成员与重建 | `StorageWorkbenchPrototype.vue` | [阵列截图](ui-design/storage-workbench/05-raid.png)、[重建截图](ui-design/storage-workbench/06-rebuilding.png) |
| quickstart 首页统一入口 | `StorageHomeSummaryPrototype.vue`、`nativeActionBridge.ts`、`NativeBusinessHost.vue` | [首页截图](ui-design/storage-workbench/00-home-entry.png)、[首页检查记录](ui-design/storage-workbench/home-integration-review.json) |

这些材料证明原型表达和浏览器演示行为，不证明真实分区、LVM、Docker 或 RAID 的操作可靠性。
