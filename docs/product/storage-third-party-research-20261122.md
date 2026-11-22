# 存储管理第三方产品调研：日常运维与故障恢复

调研日期：2026-11-22。范围是当前 quickstart 的磁盘、分区、系统应用 LVM 与 RAID 设计。
本次只记录官方资料及设计建议，不修改现有 RAID 业务，也不把演示通过当成真实存储验收。
当前设计边界见[场景与实现说明](storage-scenarios-and-implementation.md)及
[RAID 业务复用约定](raid-existing-business-integration.md)。

资料版本说明：TrueNAS 使用明确版本 25.04 的资料，Cockpit 使用可核实的 RHEL 7 官方界面说明；
它们用来研究交互模式，不宣称是最新版本。Proxmox 部分在线手册返回 403，改用官方
`proxmox/pve-docs` 仓库原文。动态网站返回不完整时，使用文档自身指向的官方源码，未以论坛替代证据。

## 1. 值得参考的四类产品

| 产品 | 已核实的产品事实 | 可采用的设计判断 | 不直接照搬 |
| --- | --- | --- | --- |
| Synology Storage Manager | 统一管理硬盘、池与卷，支持成员替换、维护速度控制、S.M.A.R.T. 与数据清理；能力受级别、型号和介质限制。[官方规格](https://www.synology.com/en-global/dsm/7.4/software_spec/storage_management) | 首页异常摘要直接定位到盘或阵列；维护入口放在对应对象详情，能力不足说明原因 | SHR、Btrfs 自修复和厂商兼容名单不属于当前 mdadm + EXT4 能力；不能据此禁止所有混合接口 |
| TrueNAS | 换盘先核对故障盘身份，非热插拔设备需要关机；替换盘有容量要求，替换后重建并最终恢复在线。[换盘手册官方源码](https://raw.githubusercontent.com/truenas/documentation/25.04/content/SCALETutorials/Storage/Disks/ReplacingDisks.md) | 将“识别故障盘 → 确认物理操作 → 选择目标 → 加入 → 重建完成”表达为可跟踪过程 | ZFS 的 offline / replace / resilver 命令及 Force 路径不能套给当前 RAID 后端；不新增绕过保护按钮 |
| Proxmox VE | 存储区分内容类型、卷身份及业务归属，thin 存储满会影响其中的客体。[存储模型原文](https://raw.githubusercontent.com/proxmox/pve-docs/master/pvesm.adoc) | 一个资源由一个管理者写入；页上展示物理来源、消费者和真实剩余量 | 共享配置不等于共享数据；不引入集群、Ceph 等本项目无需求的复杂度 |
| Cockpit / RHEL Web Console | 从 VG 进入 LV，区分卷与文件系统；创建、格式化、挂载及开机挂载有各自状态和步骤。[官方存储管理说明](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/7/html/managing_systems_using_the_rhel_7_web_console/managing-storage-devices-in-the-web-console_system-management-using-the-rhel-7-web-console) | 用户看一个简洁操作，执行结果按实际层次表达；“可用”必须包含文件系统与挂载结果 | 不因为别的系统提供 shrink 就开放当前布局缩容；需按目标系统文件系统和工具资格验证 |

以上“可采用”均为本项目建议，不是厂商对 quickstart 实现的承诺。

## 2. 最值得补齐的交互

### 2.1 能从屏幕准确找到实物硬盘

TrueNAS 换盘手册明确要求非热插拔流程先查序列号，并在取出后再次核对；容量相同不足以证明拿对了盘。
[官方换盘说明](https://raw.githubusercontent.com/truenas/documentation/25.04/content/SCALETutorials/Storage/Disks/ReplacingDisks.md)

**本项目建议：** 硬盘详情默认显示型号、序列号末段、容量、当前接口及自定义位置备注。
“查看硬盘位置”打开身份卡，不仅跳到 `/dev/sdX`。只有硬件能力确认时才显示定位灯和热插拔提示；
没有灯控时允许备注“机箱左侧第二块”，并显示完整序列号供核对。缺失或重复身份应明确标记。
首次识别信息不能替代执行前复核，也不能把用户备注当成硬件身份。

### 2.2 健康状态需要测试、结果与时间

TrueNAS 提供手动及计划 S.M.A.R.T. 测试、进度和测试结果；不兼容设备会单独列出。
文档建议避开与 scrub 或其他重负载保护任务重叠。[S.M.A.R.T. 测试说明](https://www.truenas.com/docs/scale/25.04/scaletutorials/dataprotection/smarttestsscale/)

**本项目建议：** 详情增加“上次检测 / 结果 / 下次检测”，复用现有 S.M.A.R.T. 测试、结果、属性与计划配置接口
（`backend/modules/smart/{commands,config}/`）；本项是统一入口、状态语义与维护调度衔接，不是从零实现检测。
具体硬盘支持的测试类型仍需查询设备能力。健康数据读不到时显示“未知 / 不支持”，
不能沿用最后一次绿色状态而不显示时间。耗时检测显示“可离开页面”，且不要与重建同时启动重负载扫描。

### 2.3 不打开网页也能知道出问题

Proxmox 的通知事件带时间、严重程度、类型等元数据，匹配器将其送到通知目标；
文档还明确指出 SMTP 目标没有失败排队重试机制。[官方通知原文](https://raw.githubusercontent.com/proxmox/pve-docs/master/notifications.adoc)

**本项目建议：** 本地事件记录作为事实来源，网页展示只是一个渠道。
关键事件包括成员丢失、重建失败、池容量 / 元数据紧张、开机挂载失败、备份失败。
先接一个已有通知渠道和“发送测试通知”，再考虑复杂规则编辑器。
确认收到不等于修复；恢复正常应另记事件。发送失败也要可见，不能把调用发送函数成功当送达。

### 2.4 “扫描恢复”先让用户看懂发现了什么

TrueNAS 的导入入口会列出已存在但未连接的池，再由用户选择导入；它明确只适用于 ZFS 池，
且源机器仍在使用会导致问题。[导入池官方源码](https://raw.githubusercontent.com/truenas/documentation/25.04/content/SCALETutorials/Storage/ImportPoolScale.md)

**本项目建议：** 在现有 RAID 扫描恢复能力周围补“识别结果”展示：阵列 UUID、级别、找到 / 缺少的成员、
当前是否已由本机使用、是否冲突。扫描识别、组装恢复、强制恢复应在语义上分开；
目前后端若是立即改变配置，界面应如实说明，不能将它标成“只读扫描”。
前后端分离这两阶段是待评估的业务改进，不应在 UI 重做时偷偷改变现有 Autofix。

### 2.5 扩容结果要区分“阵列变大”和“文件能用”

Cockpit 的官方流程把 LV 与文件系统分开，并明确格式化后还需挂载、开机挂载另有设置。
[官方 Web Console 存储管理说明](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/7/html/managing_systems_using_the_rhel_7_web_console/managing-storage-devices-in-the-web-console_system-management-using-the-rhel-7-web-console)

**本项目建议：** 扩容结果列出已验证的实际阶段，例如“成员已加入 / 阵列已扩大 / 文件系统尚未扩大”。
主按钮为当前能做的下一步；不能只把块设备新尺寸写回“文件可用容量”。
LV / 文件系统 / 应用分别验证；自动续做必须以实际支持的布局为前提。
这可以继续复用已有 Add 接口，再独立设计文件系统扩展用例。

### 2.6 恢复点要说明范围、保留与恢复后果

Proxmox 的备份文档区分客体一致性方式；运行中 VM 可通过 guest agent freeze / thaw 改善一致性。
容器设备和 bind mount 不在其备份覆盖范围内，临时快照也有删除步骤。
[官方备份原文](https://raw.githubusercontent.com/proxmox/pve-docs/master/vzdump.adoc)

**本项目建议：** LightHost 或 Docker 恢复点至少显示“包含哪些卷 / 未包含哪些外部目录、创建时间、
一致性方式、归属应用、保留策略”。恢复时明确覆盖范围与停机；备份页显示目标、上次成功、失败原因，
并提供恢复演练路径。不能把一次薄卷快照描述为所有 Docker 数据或完整 VM 已获保护。

### 2.7 耗时操作要有持久结果和维护窗口

Synology 支持池维护速度调节；TrueNAS 的测试说明强调重负载维护任务应错开。
[Synology 官方规格](https://www.synology.com/en-global/dsm/7.4/software_spec/storage_management)、
[TrueNAS 测试说明](https://www.truenas.com/docs/scale/25.04/scaletutorials/dataprotection/smarttestsscale/)

**本项目建议：** 仍用现有四个工作区，在“任务记录”展示操作结果，在对象详情展示内核重建状态；
两者互相关联。显示运行阶段、最近更新时间及可获得的进度 / 速度；无可靠进度时显示处理阶段，
不制造百分比或完成时间。速度调整、暂停仅在现有后端有明确支持和恢复语义时开放。

## 3. 不可忽略的技术边界

| 边界 | 官方依据 | 本项目的落地建议 |
| --- | --- | --- |
| thin 是块存储且依赖池真实空间 | Proxmox LVM-thin 支持快照 / 克隆，按写入分配，并是本地存储。[LVM-thin 原文](https://raw.githubusercontent.com/proxmox/pve-docs/master/pve-storage-lvmthin.adoc) | 同时监测逻辑上限、池数据、元数据及 VG 可扩展空间；展示跨卷影响，不以“不超额分配”替代满池保护 |
| 不能多处注册同一底层数据而误以为独立资源 | Proxmox 警示同一存储的别名配置可能使不同 ID 指向同一映像，并说明卷归属关系。[存储模型原文](https://raw.githubusercontent.com/proxmox/pve-docs/master/pvesm.adoc) | 全局库存按实际身份去重；LightHost / Incus 与系统应用保留管理边界；跨入口操作共用身份与依赖判断 |
| USB 身份不一定可靠，SMR 不能只按容量判断 | TrueNAS 硬件指南警示 USB 可能错误报告序列号，并不建议其 ZFS 主存储使用 SMR。[硬件指南](https://www.truenas.com/docs/scale/25.04/gettingstarted/scalehardwareguide/) | 采集接口、身份可信度、扇区、容量与已知介质特征；按当前工具链核验。SMR 未知就显示未知；不把 TrueNAS/ZFS 限制直接升级为所有 Linux RAID 的禁令 |
| 介质与布局兼容规则是产品特定的 | Synology 列出池的接口 / 介质 / 4Kn 混用限制，并说明功能依型号与级别变化。[官方规格](https://www.synology.com/en-global/dsm/7.4/software_spec/storage_management) | 替换准入在服务端按实际成员需求、可用扇区及工具能力计算，不以标称 TB 或阵列总大小比较；规则需目标平台验证，不复制厂商白名单 |
| RAID 一致性检查不等于文件内容自修复 | Synology 将文件系统清理与 RAID 清理分开，并限定适用结构。[官方规格](https://www.synology.com/en-global/dsm/7.4/software_spec/storage_management) | mdadm + EXT4 的维护名称和结果保持准确；没有内容校验依据就不宣称能判断哪份文件数据正确 |
| 开机可用是独立验收维度 | Cockpit 明确区分格式化、挂载和开机挂载选项。[官方说明](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/7/html/managing_systems_using_the_rhel_7_web_console/managing-storage-devices-in-the-web-console_system-management-using-the-rhel-7-web-console) | 核验 md 组装 → LVM 激活 → 文件系统挂载 → Docker / LightHost 启动；依赖未就绪时不能把应用写入空挂载目录当正常 |

除产品交互参考外，Linux md 的技术事实也必须单独确认：内核文档说明 `check` 在某些级别可能发生修复，
因此不能笼统命名为“绝不写盘的校验”；dirty 且 degraded 的 RAID 5 / 6 通常拒绝启动有其数据一致性原因。
[Linux md 官方文档](https://docs.kernel.org/admin-guide/md.html)

## 4. 保持当前界面简洁的实施顺序

**先补底座与真实信息：** 可靠身份、原始与派生状态、物理位置、依赖与所有者、真实容量、开机挂载结果。
这些进入现有卡片和右侧详情，不新增一个大型设置中心。

**再补运维闭环：** 可续看的长任务、异常时间线与一个通知出口、扫描恢复识别结果、扩容分阶段结果。
复用已有业务操作；需要改变后端流程的部分独立评审和验收。

**随后补可维护性：** 健康检测计划、已有 RAID 的一致性维护、恢复点保留与实际回收结果、异盘备份和恢复演练。
每项能力须有目标系统支持和故障场景验证，不能只增加按钮。

暂不以此次调研为由引入 ZFS、Ceph、任意跨盘 VG、缓存分层、集群存储或厂商级热备自动化。
已有 RAID 六级别和现有业务接口也不因调研建议被缩减。
