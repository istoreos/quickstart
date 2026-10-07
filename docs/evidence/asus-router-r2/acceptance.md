# ASUS 网络设备呈现修正验收

- 产品语义：继续要求型号、主机名、网关/AP 或拓扑证据才能自动判定网络设备；不把所有 `ASUSTeK COMPUTER INC.` 默认判为路由器。
- 用户确认设备：`mac:60:cf:84:21:b7:a0`（`60:CF:84:21:B7:A0`）人工设置为 `network`，作用域为 `persistent`。
- quickstart 重启后验证：类别仍为 `network`，来源仍为 `manual`。
- 无名称设备呈现：主标题为“ASUS 网络设备”，第二行仅显示“手动设置”，不再重复“ASUS · 网络设备”。
- 图标：`/luci-static/quickstart/device-icons/network.webp`，为原创通用路由器形态，不使用 ASUS Logo。
- 桌面 1440×900 与移动端 390×844 验证通过；移动端无页面级横向溢出。
- 前端测试 24 项、类型检查和生产构建通过；资源缓存版本为 `0.14.0-r2`。
- 测试机文件级备份：`/root/quickstart-backups/asus-presentation-r2-20260924`。

机器可读结果与截图位于本目录。该人工分类是用户确认后的目标配置，因此验收结束后保留。
