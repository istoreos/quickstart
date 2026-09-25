# M18 验收记录：设备分组、优先级与定时规则

> 日期：2026-09-24；测试机：`root@192.168.9.215`；备份：`/root/quickstart-backups/m18-20260924120000`

## 任务树完成情况

- [x] 独立 Device Group 存储，和 DHCP 地址分配标签完全隔离。
- [x] 全局 → 低优先级组 → 高优先级组 → 单设备的确定性策略合并与来源解释。
- [x] 后端按路由器时区执行星期/跨午夜日程，返回下次检查时间。
- [x] 服务启动即创建单一有界分钟循环，不依赖用户先打开页面；复用现有 access/speed/route 模块，依赖缺失写入有界降级事件。
- [x] 全局策略从 Device Inventory 取得最多 2048 个稳定身份，真正覆盖未加入分组的设备；移除全局/分组/计划时恢复默认值。
- [x] 相同有效策略按签名跳过重复写入，计划结束和移除成员会执行一次显式恢复。
- [x] 家庭分组、成员搜索、联网设置、休息时段的响应式 UI。

## 验收结果

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| 儿童组与跨午夜阻断/恢复 | 通过 | `TestDeviceGroupScheduleWeekdayAndCrossMidnight` |
| 多组优先级、全局与设备例外 | 通过 | `TestDeviceGroupPolicyPrecedenceAndExplanation` |
| 重启持久化、删除组保留设备例外 | 通过 | 单元测试及 `api-check.cjs create → restart → verify` |
| 128 组、2048 成员、512 策略、256 事件 | 通过 | `TestDeviceGroupCapacityAndEventBounds` |
| 不创建每设备 goroutine | 通过 | `TestDeviceGroupReconcileUsesBoundedSinglePass` |
| 重启自动调度、全局覆盖及规则结束恢复 | 通过 | 构造时启动唯一调度器；`TestDeviceGroupGlobalPolicyAppliesToInventoryAndRestores`、`TestDeviceGroupScheduleRestoresAndRemovedMemberIsReset` |
| API 版本冲突与原子写入 | 通过 | JSON 原子 rename、version hash、路由测试 |
| 桌面/紧凑/移动视口 | 通过 | `desktop-1440x900.png`、`compact-1024x768.png`、`mobile-390x844.png` |
| 全量/竞态/前端/生产构建 | 通过 | Go `./...`、定向 `-race`、37 个前端测试、`vue-tsc`、Vite build |

## 真机结果与恢复

- 后端 SHA-256：`3722b526ccb0425e964bde4119b3341a5ee0ca493e241c0b6db57f15d469729b`。
- Web 缓存版本：`0.14.0-r11`。
- DHCP/network/firewall 哈希保持为 `e158cc…` / `86fdfc…` / `7f7019…`。
- 验收临时分组已删除；测试前设备分组文件不存在，验收后已恢复为不存在。
- UI 首轮发现 LuCI 全局标题样式和命中层级冲突，经两次优化后，三种视口可点击且无横向溢出。
