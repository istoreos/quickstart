# 设计预览与浏览器检查工具

本目录保留仍被 `scripts/ops/` 调用的浏览器检查脚本，以及预览辅助文件。
当前 Vue 预览入口为 `web/src/design/` 与 `web/vite.design.config.ts`，运行方式见
[开发指南](../../development.md)。

当前存储原型入口为 `/storage?variant=A`，已接到 quickstart 原型首页的「硬盘与存储」导航、
磁盘摘要卡片及原有磁盘操作入口。`/nas` 是新入口别名；上一轮单空间设置稿保留在
`/design/storage-setup?variant=A`。它的用途向导被直接复用到新页面，不形成第二套真实磁盘后端。

### 多盘工作台

- A 多盘布局（推荐）：所有硬盘的分区条＋点选后从右侧滑出的区域详情。B 逐盘管理。C 按用途查看。
- 详情抽屉沿用局域网管理的交互：桌面宽 440px，手机全宽；顶部关闭、底部返回，内容独立滚动。
  关闭后保留列表位置与选中项。取消操作回到详情，加入计划后收起详情，继续检查待执行列表。
  [桌面效果](storage-workbench/10-detail-desktop.png)、[手机效果](storage-workbench/11-detail-mobile.png)、
  [抽屉检查记录](storage-workbench/drawer-review.json)。
- 硬盘与分区：系统区保护、普通 EXT4 挂载、在未分配空间中创建分区、格式化和删除未使用分区。
- 存储池与卷：来源分区 → VG → thin-pool → 卷；分别展示逻辑容量、数据占用、元数据与余量。
  系统池可演示创建、扩容、删除未使用卷；Incus 管理的卷只提供对应管理入口。
- RAID：卡片展示阵列，右侧详情管理，创建与成员变更采用“选择 → 核对 → 执行”的向导。
  保留创建、扩充、移除、恢复、详情、删除和扫描恢复；原处理函数、级别、成员规则、禁用条件和接口继续复用。
  页面内影响确认替换旧浏览器双确认，详见[业务复用与新交互](../raid-existing-business-integration.md)。
- 分区和 LVM 布局变更先加入待执行列表。RAID 保留原操作流程，不进入此列表。
  可以撤销未应用的计划；最终应用再次确认清空目标。失败停止后续操作，逐项记录，
  不把多项操作描述为可以整体自动回滚的事务。
- 首页和工作台共享同一份内存模型。回到首页、再次进入页面不会丢失演示操作；刷新重置示例。

演示场景：`/storage?scene=raid&tab=raid`、`/storage?scene=degraded&tab=raid`、
`/storage?scene=raid-unmounted&tab=raid`、`/storage?scene=raid-resyncing&tab=raid`、
`/storage?scene=raid-missing&tab=raid`、`/storage?scene=thin-full&tab=pools`。
所有布局和容量均为示例，不调用真实磁盘、LVM、mdadm 或 Docker。

### 现有网络存储入口

“磁盘阵列”归并到 `/storage?tab=raid`；SMART 保留 `/smart`、`/smart/task`、`/smart/log`，
复用原组件与业务处理，设计入口使用卡片和手机适配。菜单呈现“硬盘与存储 / S.M.A.R.T.”。
[整合决策、只读真机核对与业务边界](../storage-existing-menu-integration.md)。

### 桌面与手机导航

完整效果版：首页 `/?presentation=1`，存储 `/storage?presentation=1`。
按用户真机截图补了可操作的宿主评审外壳：品牌顶栏、系统侧栏、搜索、“网络存储 → 硬盘与存储”以及手机菜单收起。
效果版隐藏评审工具，普通地址仍可选择演示场景。
该外壳只在 design 入口运行，不改真机主题或菜单注册；详见下方规则文档。

按真实 iStoreOS 手机截图修订：复用现有系统侧栏；删除原型额外的“首页 / 硬盘与存储”顶栏及存储局部左栏。
页内使用横向工作区标签，桌面完整名称，手机简化为“硬盘 / 存储池 / RAID / 任务”。
提醒置于标签行右端，任务仅在进行中时有角标；保持 `?tab=` 和浏览器前进/后退同步。
[导航规则与真实宿主边界](../storage-responsive-navigation.md)、
[桌面截图](storage-workbench/navigation-desktop.png)、[手机截图](storage-workbench/navigation-mobile.png)、
[导航检查](storage-workbench/navigation-review.json)。

### 日常维护与异常闭环（2026-11-22）

[目标与任务树](../storage-implementation-goals-20261122.md)逐项追踪 S1–S9；
[生产技术契约](../storage-production-contract-20261122.md)拆分 B01–B16 及 H1–H8 故障验收。
新增入口保留在四个工作区和右侧抽屉中：

- 硬盘详情 →「健康与物理身份」「名称与诊断资料」；支持 SMART 演示任务、结果、计划和用途别名。
- 存储池 →「容量与空间处理」「应用接入与挂载检查」；卷行提供恢复点归属和新卷接入说明。
- RAID →「识别已有阵列」「健康与维护」「换盘与重新加入」；识别只读，不调用旧 Autofix。
- 顶部「存储提醒」→ 告警已读/恢复/复发、对象定位、模拟通知测试；首页沿用同源异常摘要。
- 任务记录 → 阶段结果、部分完成、查询中断与续看；不把断网当任务失败，不虚构实时百分比。

可直接选择页面顶部场景，或使用：
`/storage?scene=read-error`、`/storage?scene=foreign`、`/storage?scene=identity-unknown`、
`/storage?scene=app-inuse`、`/storage?scene=mount-failed&tab=pools`、
`/storage?scene=partial-grow&tab=jobs`、`/storage?scene=resume-task&tab=jobs`。

新交互全部使用内存示例。刷新重置；离开工作区/SPA 返回保留本次记录。
恢复选定阵列、原盘专用重新加入、真实扩池、恢复点写入和备份接口未接入时明确禁用，
不以旧 Autofix 或提示消息替代尚不存在的能力。后台持久任务、实际外发、开机守卫和真机故障验收仍待生产阶段实施。

检查命令（预览先启动；Playwright 与浏览器路径按本机设置）：

```bash
rtk proxy /var/cache/build-cache/run.sh -- node scripts/review-storage-model.cjs
rtk proxy env PLAYWRIGHT_MODULE=/path/to/playwright PLAYWRIGHT_CHROMIUM=/path/to/chromium node scripts/review-storage-care.cjs
```

最终结果及逐轮记录见目标文档，不用历史证据替代当前验收。

### 交互参考与取舍

完整的[参考记录](../storage-interaction-references.md)保存来源、采用方式与不沿用的能力；
[日常场景与实现边界](../storage-scenarios-and-implementation.md)逐项检查操作后的去向、失败恢复、
quickstart 整合方式和正式接入前需要补齐的环节。原型演示通过不等于真实存储生命周期已经闭环。
进一步对照成熟产品的[官方资料调研](../storage-third-party-research-20261122.md)和
[差距与实施方案](../storage-gap-analysis-20261122.md)记录健康维护、恢复识别、告警及真实后端细节。

- [DiskGenius 创建分区](https://www.diskgenius.com/manual/create-new-partition.php)：借鉴分区图和先规划、再保存；
  本原型不继承其无损移动、拆分或恢复分区等能力声明。
- [Windows 磁盘管理](https://learn.microsoft.com/en-us/windows-server/storage/disk-management/overview-of-disk-management)：
  借鉴多盘同屏和未分配空间的明确区分；已有文件系统空闲容量不当作未分配区域。
- [Synology Storage Manager 总览](https://kb.synology.com/en-uk/DSM/help/DSM/StorageManager/overview)：
  借鉴健康优先、成员状态及池与卷关联，RAID 警告可以直接进入处理位置。
- [RHEL / Cockpit 存储管理](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/7/html/managing_systems_using_the_rhel_7_web_console/managing-storage-devices-in-the-web-console_system-management-using-the-rhel-7-web-console)：
  参考物理设备、RAID 与 LVM 的关系展示。这里使用自己的管理归属及依赖保护模型。

普通文件存储仍是默认路径。已有 EXT4 可保留文件挂载；重新规划会清空所选范围，不承诺无损转换。
普通 EXT4、LightHost 主机存储、系统应用存储可以并存。沙箱卷仍标记待支持。

通过 `./scripts/ui-design.sh start|status|stop` 管理；脚本启动时打印存储原型地址。
默认 5180；可使用 `QUICKSTART_UI_DESIGN_PORT=5181`，各管理命令使用相同环境变量。
正式前端入口不导入原型。首页磁盘卡片替换通过 design Vite transform 完成，生产组件不被改写。

FastNet「外网测速」卡片见 [产品交互说明](../fastnet-card-design.md)。启动预览后访问
`http://127.0.0.1:5180/`，默认先安装；右下「卡片预览」切换状态。
`/?external=present&scene=ready` 展示已有结果，`&form=summary` 展示摘要。
预览使用正式首页、统一三态主题和独立构建的 FastNet 外部组件，只模拟服务数据与安装反馈。
`fastnet-card-review.cjs` 委托当前 `scripts/ops/fastnet-card-review.cjs`，需要设置 `OUTPUT_DIR`。
真实安装、接口、后台任务和资源测量见 [实施里程碑](../../archive/product/fastnet-card-implementation-milestones.md)。

多局域网设计提案见 [产品与技术方案](../multi-lan-design.md)。启动预览后访问
`http://127.0.0.1:5180/design/multi-lan`，可切换多 LAN、没有默认 LAN 和单 LAN 服务异常场景。
交互检查入口为 `multi-lan-review.cjs`，需要可用的 Playwright；可用 `CHROMIUM_PATH` 指定已安装的浏览器，
`OUTPUT_DIR` 指定截图与检查结果目录。该页面是独立设计提案，不代表正式版本已支持多 LAN。

过去的设计说明与逐轮评审已移入 `docs/archive/product/`；历史输入和输出位于
`docs/archive/evidence/`。部分检查需要 Playwright、已构建的 Web 资源或显式传入的测试设备参数，
不属于默认本地测试，也不能作为当前设备状态说明。

部分早期评审仍绑定历史规格版本，例如 `structure-spec-review.cjs` 要求布局规范 1.4，
而归档规范已更新为 1.7。这些脚本供追溯，不是当前版本的验收入口。
