# M13 验收记录：独立设备资料

> 日期：2026-09-24；测试机：`root@192.168.9.215`

## 任务树完成情况

- [x] 建立独立 `DeviceProfile` Get/Patch/Reset 接口，设备备注名不再复用 DHCP Hostname。
- [x] 持久 MAC 身份保存 Unicode 备注名和人工分类；boot-local 身份仅在本次开机有效。
- [x] 旧分类覆盖自动迁移到 version 2 资料存储，原子写入且容量上限为 2048。
- [x] 原始 DHCP Hostname、厂商和识别依据保持只读；损坏存储降级为 partial 而不隐藏设备。
- [x] 详情内完成改名、类型修正和恢复自动，列表同步更新。

## 验收结果

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| 中文备注名与 DHCP 配置隔离 | 通过 | Profile 定向测试；真机改名时 DHCP/network/firewall 哈希不变 |
| 持久身份重启保留 | 通过 | `restart-check.cjs` 写入 → Quickstart 重启 → 读取 → 恢复原值 |
| boot-local 不跨重启持久化 | 通过 | `TestDeviceProfileBootAliasIsNotPersisted` |
| 迁移、2048 上限、并发原子写入 | 通过 | `TestDeviceProfileStore*`、`TestDeviceProfileConcurrentUpdatesRemainReadable` |
| 损坏文件安全降级 | 通过 | `TestCorruptDeviceProfileStoreDegradesInventoryWithoutHidingDevices` |
| 三档响应式和中文布局 | 通过 | `desktop-1440x900.png`、`compact-1024x768.png`、`mobile-390x844.png` |

真机验收结束后已恢复原备注名，没有创建静态租约或触发 dnsmasq reload。M13 达成，无需产品语义调整。
