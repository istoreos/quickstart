# 局域网设备管理入口与一次性迁移契约

> 状态：M23 冻结；服从[当前唯一产品与领域基线](./lan-device-management-product-architecture.md)  
> 日期：2026-09-25

## 唯一入口映射

| 旧入口或能力 | 唯一新入口 | 保留的用户结果 |
| --- | --- | --- |
| 设备列表 / 经典设备列表 | 设备 | 查找、筛选、查看状态、进入设备详情 |
| 单独静态地址列表 | 设备 → 网络与上网；离线和批量规则进入局域网设置 → 规则台账 | 地址预留、冲突处理、离线维护、批量删除 |
| 单独限速设备列表 | 设备 → 使用限制；离线和批量规则进入局域网设置 → 规则台账 | 限速、联网权限、离线维护、批量删除 |
| DHCP 标签编辑 | 局域网设置 → 上网路线 | 路线目标、引用数量、替代后删除；普通用户不输入 tag |
| 分组面板 | 分组与计划 | 分组、共同规则、时段、额度、来源解释 |
| 全局 DHCP | 局域网设置 → 地址分配 | DHCP 状态、地址池、关闭影响与恢复 |
| 浮动网关表单 | 局域网设置 → 上网路线 → 浮动网关 | 配置、持有者、健康、演练和引用 |
| 全局限速 | 局域网设置 → 带宽与统计 | 总带宽、统计来源和能力状态 |
| 高级网络工具 | 局域网设置 → 诊断与高级 | 审计、探测、通知和策略备份 |

规则台账不是一级入口，也不复制设备详情的日常编辑器。它只服务离线规则、孤立规则、冲突定位和批量维护。

## 迁移方式

迁移采用 `adopt_in_place`：运行配置仍由 OpenWrt 原生配置文件提供事实，新 V2 Module 将可表达的既有规则映射为 Device Profile、Device Network、Device Restrictions、Gateway Target 或规则台账记录，并写入一份版本化迁移回执。它不复制一套配置，也不建立双写。

流程固定为：

1. `GET /cgi-bin/luci/istore/lanctrl/v2/migration/plan/` 只读现有规则和配置哈希；
2. 报告每项 `adopt`、`normalize`、`conflict` 或 `unresolved`，保留中文旧名称和孤立规则；
3. 有冲突或未知项时只输出完整报告，不允许 apply；
4. `POST /cgi-bin/luci/istore/lanctrl/v2/migration/apply/` 必须提交预览版本；
5. apply 原子写入 `/etc/quickstart/lan-device-model-v1.json` 回执并重新读取验证；
6. 验证失败恢复旧回执；恢复也失败则返回 `recovery_required`；
7. 相同源版本重复 apply 不写盘，配置改变后必须重新预览。

迁移回执不是第二份策略配置，只记录模型版本、源版本和旧规则到新用户结果的映射。实际规则继续由新的深 Module 通过 OpenWrt 原生配置读取和写入。

## 退场清单

以下对象在 M30 删除；M23～M29 期间只允许为迁移、测试或内部适配保留，不得承载新增产品能力：

- `deviceList.vue` 经典列表切换；
- `staticStateList.vue` 和 `speedLimitList.vue` 一级路由；
- 普通界面中的 DHCP tag、tag title 和 option 3/6 编辑；
- `/lanctrl/staticDeviceConfig/`、`/lanctrl/speedLimitConfig/`、`/lanctrl/dhcpTagsConfig/` 等旧写 Interface；
- 前端直接拼装 UCI、eqos 或 floatip 字段的 store；
- `device_inventory_v2` 作为新旧页面切换开关。

读取旧配置只属于迁移 Adapter。正式 UI 完成后，所有用户写入只通过 Device Profile、Device Network、Device Restrictions、Gateway Policy、Groups & Plans 和 LAN Settings Interface。
