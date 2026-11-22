# 现有网络存储菜单整合

日期：2026-11-22。用户截图含“磁盘阵列”和“S.M.A.R.T.”。
本轮只读连接 `root@192.168.30.7`，核对 `/usr/lib/lua/luci/controller/quickstart.lua`：
现有 RAID 菜单位于 `admin/nas/raid`（order 10），SMART 位于 `admin/nas/smart`（order 11）。
未修改该设备的配置、菜单或软件。

## 最终菜单

```text
网络存储
├─ 硬盘与存储
│   ├─ 硬盘与分区
│   ├─ 存储池与卷
│   ├─ RAID 阵列（原“磁盘阵列”）
│   └─ 任务记录
└─ S.M.A.R.T.（硬盘健康快捷入口）
    ├─ 常规设置
    ├─ 计划任务
    └─ 查看日志
```

四个存储工作区仍是页内标签；SMART 保留旧名称和熟悉的三个管理区域。
RAID 是存储结构的一种，纳入工作台能让成员、文件系统与用途关联；
SMART 面向物理硬盘健康，既供整机管理，又能从单盘详情进入，不增加第五个存储页签。

## 业务复用与旧入口

- “磁盘阵列”不重复出现在新系统菜单中，原功能位于“硬盘与存储 → RAID 阵列”。
  保留已有创建/扩充/移除/恢复/详情/删除/扫描恢复的处理和接口。
- 预览 `/raid` 重定向 `/storage?tab=raid`，保留演示参数；页面中旧 LuCI RAID 链接也定位此页。
  真机旧 URL 和菜单注册的兼容跳转尚待正式接入，本轮没有发布到设备。
- SMART 预览接入现有 `pages/smart` 与 `components/smart`，保留原请求和保存处理函数。
  `/smart`、`/smart/task`、`/smart/log` 对应三个已有区域；设备详情保留信息/属性/自检日志/扩展信息。
- 仅 design Vite 改呈现：设备卡片先展示，全局监测配置折叠；主题、字体与手机弹窗一致。
  原温度阈值、电源模式、四种计划类型及日程字段不删减。
- SMART mock 只更新内存。测试提交明确不等于通过；不存在实际 SMART 结果时显示未知。
  新配置、原始属性、结果和日志均不发送到真实设备。
- 原有 SMART 弹窗工厂的移除 DOM 不触发卸载；预览适配先 `render(null, el)`，清理轮询与滚动锁。
  这是 design 适配，不表示生产代码已修复该生命周期问题。

## 能力边界

旧 SMART 代码的温度配置加载、空/错误响应等仍需生产回归；例如父视图 setConfig 没有完整回填温度阈值，
不能因新 UI 接入而宣称原实现已完成全部状态改造。
本次浏览器走查还复现了未配置设备的首次编辑分支：`onEditDisk` 在 tmpIndex 为 -1 时未加入用户编辑值，
随后初始化设备列表使用全局阈值，导致第一次保存回落到全局阈值；已配置设备分支可按编辑值保存。
该问题作为既有业务缺陷记录，测试明确保留已知问题状态，不把它包装成新 UI 完全通过。
存储详情的简化维护演示与全局 SMART 设置各有示例状态，正式实现必须由同一个 SMART 服务供数，
不能把两份浏览器示例状态当成两个配置中心。
本轮没有修改正式 SMART/RAID 组件、请求文件或后端。

预览入口：`/storage?presentation=1&tab=raid`、`/smart?presentation=1`。

## 本轮检查

- [整合检查与原 SMART 请求参数](ui-design/storage-workbench/existing-menu-review.json)：8 组走查，状态为 `passed-with-known-legacy-issue`。
- [手机菜单](ui-design/storage-workbench/existing-storage-menu.png)、[手机 SMART](ui-design/storage-workbench/smart-mobile.png)、[桌面 SMART](ui-design/storage-workbench/smart-desktop.png)。
- 宿主导航 8 组与原 RAID 13 组回归通过；类型检查与 design 构建通过；没有真实写请求。
- “已配置设备编辑”与“首次配置设备”明确分开验证；后者缺陷未修复，保留在检查结果的 knownLimitations。
