# RAID 接入存储工作台：业务复用，交互重新设计

更新日期：2026-11-22。本次用户澄清：产品能力、代码与业务沿用已有 RAID，交互和 UI 可以修改，
并且应融入 quickstart 当前主题。此决定替代上一版“原页面、操作与确认形式一模一样”的理解。

## 业务与界面的边界

业务基线仍是 `web/src/pages/raid/index/index.vue`、`web/src/components/action-raid/`、
`web/src/request/request.ts` 中已有的处理函数，以及现有 RAID 后端。
创建、扩充、移除、恢复、详情、删除、扫描恢复全部保留；级别、默认值、最少成员数、候选来源、
禁用条件、请求参数、错误和结果处理均以已有实现为准。没有另写 RAID 引擎或更换后端。

界面不再搬用旧表格和旧弹窗。用当前存储工作台的卡片、间距、颜色、按钮和右侧详情表达同一套业务。
分区 / LVM 的待执行列表仍独立保留，RAID 沿用直接发起操作的业务，不增加新的 RAID 排队协议。

## 本轮交互

| 场景 | 新界面与交互 | 保持的业务 |
| --- | --- | --- |
| 阵列概览 | 一阵列一卡片，突出状态、容量、成员和文件位置；原始状态可在详情查看 | 列表接口和原轮询；容量、成员、挂载信息取自返回结果 |
| 管理阵列 | 点“查看与管理”右侧滑出；完整操作按用途排列，手机全宽 | 原操作入口和禁用规则全部保留 |
| 创建 | 六种级别卡片、硬盘选择卡片；选择配置 → 核对影响 → 创建 → 结果 | 默认 RAID 5，原级别和最少成员数、候选、Create 参数与结果处理 |
| 扩充 / 移除 / 恢复 | 选择一块硬盘 → 核对目标与影响 → 原操作执行 | 三个接口保持独立，不把恢复当扩容，不把移除当缩容 |
| 删除 | 先处理挂载；未挂载时核对影响，输入阵列名称并勾选确认 | 先卸载条件、原 Delete 参数及有数据阵列删除能力 |
| 扫描恢复 | 简洁确认面板及原扫描反馈 | 原 Autofix 回调与刷新 |
| 技术详情 | 只读信息面板，可关闭回到原详情侧栏 | 原 Detail 返回文本 |

创建和删除的浏览器双重 confirm 改为页面内的明确影响确认。
新界面完成确认后，仍由原处理函数执行原有校验和请求。返回修改保留选择，并清除上一步确认；
阵列成员或状态变化会使删除确认失效。错误沿用现有处理，不虚报成功。

卡片以事实展示状态：例如原始 degraded 显示“已降级”，resync 显示“同步中”，
原始状态保留在详情；显示名称的转换不改变 `getDisabled` 或接口判定。

## 操作契约

| 操作 | 已有处理位置 | 原接口与主要参数 |
| --- | --- | --- |
| 列表 | RAID 页面 | `GET /raid/list/`；原每 5 秒轮询 |
| 候选成员 | 创建 / 扩充 / 恢复组件 | `GET /raid/create/list/`；不在新前端另加“必须全空盘”过滤 |
| 创建 | `setup-create.vue` | `POST /raid/create/`：`level, devicePaths` |
| 扩充 | `setup-edit.vue` | `POST /raid/add/`：`path, memberPath` |
| 移除 | `setup-remove.vue` | `POST /raid/remove/`：`path, memberPath` |
| 恢复 | `setup-recover.vue` | `POST /raid/recover/`：`path, memberPath` |
| 详情 | `setup-info.vue` | `POST /raid/detail/`：`path` |
| 删除 | RAID 页面 | `POST /raid/delete/`：`path, members` |
| 扫描恢复 | RAID 页面 | `POST /raid/autofix/` |

原 `getDisabled` 继续控制扩充 / 移除：级别为 `raid0` / `jbod`，或状态包含 `degraded` /
`resyncing(` 时禁用。新 UI 解释禁用原因，不自行更改判断。
后端创建后仍接 NAS 格式化 / 挂载；扩充包含成员增长；恢复不能替换成扩充，也不宣称自动扩展文件系统。

## 原型实现与正式接入

`web/src/design/raidPresentationTemplates.ts` 仅由 `vite.design.config.ts` 加载，
在预览构建中替换已有组件的模板，并附加展开、向导步骤、确认、焦点等展示状态。
原业务处理函数保留，`raidWorkbenchPreview.css` 统一样式，生产组件源文件与后端未修改。

创建分支在预览模板中补传已有 success 回调，使操作成功后立即刷新列表；原轮询仍保留。
原源码此前没有向创建子组件传递该回调，这是本次展示衔接补齐，不是新的创建业务。
已有 JBOD 前端值 `jbod` 与后端 `NormalizeLevel` 接受 `linear` 的差异仍保留，预览不偷偷转换成功。

这是验证 UI 的临时方案，不应把 Vite 模板替换机制带入生产。
正式接入时可将现有处理函数整理为共享控制逻辑，供新视图调用；候选、请求和后端职责保持原位，
避免复制第二套 RAID 校验和操作代码。业务错误与接口变更单独处理。

## 预览边界与验证

`raidWorkbenchFixture.ts` 仍只在浏览器返回示例响应，并更新工作台的示例库存。
不向路由器转发 API，不执行 mdadm、格式化或真实挂载。

当前证据为 [新版交互与业务契约检查](ui-design/storage-workbench/raid-redesign-review.json)，
逐项比对七个写 / 详情接口的实际参数，并检查成员限制、失败重试、删除前卸载、手机与主题。
[概览](ui-design/storage-workbench/16-raid-overview.png)、
[右侧详情](ui-design/storage-workbench/17-raid-details.png)、
[创建向导](ui-design/storage-workbench/18-raid-create.png)、
[手机向导](ui-design/storage-workbench/19-raid-create-mobile.png)、
[深色主题](ui-design/storage-workbench/20-raid-create-dark.png)用于视觉评审。

此前 `native-raid-review.json` 和 13～15 号截图记录的是直接搬用旧页面的上一版，供追溯，
不再作为当前界面与确认形式的验收依据。真实 RAID 后端可靠性不由本轮浏览器检查替代。
