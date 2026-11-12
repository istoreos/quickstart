# FastNet 历史容错与卡片精简验收

2026-11-10，FastNet 0.8.4，本地候选与 .7 测试部署；未推送、未发布正式源。

## 目标和任务树

1. 卡片精简：删除“默认外网出口 / FASTNET ↗”整行、终态失败提示和“重试测速”按钮；历史保存错误不在卡片全局展示。
2. 精确复现 403：原 HTTP 网关 POST 返回 403 cross_origin，HTTPS 同请求进入参数校验返回 422；旧前端历史 begin 403 后，quick.error 为“历史记录请求失败 (403)”，并跳过 WAN/LAN。
3. 修复来源判断：HTTP 局域网页面没有 Fetch Metadata，代理改写 Host。历史写请求增加非简单 X-FastNet-History 标记，仅 loopback/JSON/非跨站场景可使用；不信任转发主机、不开放 CORS。
4. 历史作为附加功能：begin 失败返回空记录标识，results/finish 无记录时跳过，WAN/LAN 仍走原业务。历史失败不变成 quick.error；真实测量错误继续显示。历史页使用中性局部提示，并保留保存重试。
5. 核心存储降级：原生归档初始化不可用时，在同一个共享执行锁内调用原处理函数，保留原互斥。历史文件不可写不阻止测速。
6. 生产构建、包版本/哈希对齐、预览故障注入、真实 HTTP/HTTPS + LAN 验收、提交代码。

## 检查结果

- 快速反馈：`node --test src/composables/useFastnet.history-isolation.test.js`（在 webui-src，用 SSD wrapper）修复前断言失败，实际 quick.error 为“历史记录请求失败 (403)”；修复后九种场景通过，包括 403/503/超时/无效记录响应、results/finish 失败、LAN 单测、原生 historyError 以及真实 WAN 错误保留。
- 后端回归修复前失败：HTTP 无 Fetch Metadata 被拒绝；quick/wan/nat/ipv6 因历史存储错误返回 503，原处理函数调用数为 0。修复后通过。外部请求、跨站、同站不同源、缺标记、表单提交和 CORS 预检仍拒绝。
- `go test ./cmd/fastnet -count=1` 全部通过；前端 composables/presentation/边界检查 40 项、卡片 11 项、插件包契约 3 项通过。
- 前端生产构建和 x86_64/aarch64/armv7 静态 Go 构建完成。卡片继续按需加载，主入口较上一版更小，没有新增依赖。
- 原 LAN 核心逐字节未改：useLanSpeed.js SHA256 459cb29d437e1336b90480fea20581ca6a8d5b9f1421ad719e77b46892bdc347。六次 Ping、单次/压测调度、流式传输、服务端上传计时和速率公式保持原实现。
- scripts/ui-design.sh 真实页面：历史全接口注入 403、503 时，完整快速体验都能结束；无全局历史 alert，进入历史页才展示中性不可用提示。外部门户回复和网络数据明确使用预览 fixture，不是真实 WAN 测速。
- .7 HTTP 与 HTTPS 网关：带浏览器标记的空参数写入都返回预期 422 invalid_request（通过来源校验，未创建记录），历史读取返回 200。
- .7 真实 HTTP iframe LAN 测试成功，产生一条只有 lanPing/lanDownload/lanUpload 的成功记录；WAN/NAT/IPv6 最新值逐字段保持。具体结果见 live/browser-report.json。本轮没有额外运行真实 WAN 测速。
- 深浅色卡片截图与实际 LAN 截图保存于本目录；底部装饰/终态重试行均已删除。

## 部署与空间处理

第一次部署在创建备份时遇到 overlay 空间不足，尚未替换任何程序，没有服务重启。六份本任务较早的 FastNet 备份经过 SHA256 和 tar 完整性校验后转存到本地忽略目录 bin/fastnet-device-backups-20261110；设备保留最近的完整/界面回滚副本。删除的仅为已转存的本任务备份和本次不完整备份，没有清理其他项目文件。部署脚本增加备份前剩余空间检查。

最终部署资源版本：fastnet-square-20261110T055428Z。地址：http://192.168.30.7:10000/cgi-bin/luci/admin/quickstart 。私有回滚脚本：/root/quickstart-backups/fastnet-square-20261110T055428Z/rollback.sh。

后端修复需要且已完成一次 FastNet 重启。网络、DHCP、防火墙、FastNet 配置、首页布局/启动器、opkg 记录和 Quickstart 后端哈希均保持不变，原历史保留；后续真实 LAN 验收按预期增加一条记录。设备运行版本为 0.8.4。

候选：FastNet 0.8.4-1、LuCI 0.8.4-r1、meta 0.8.4-13。运行归档 SHA256 6743126f3b4588fe70f5fc63fe301e68d036a2c19cad3437dd677aa382fcb769，与运行包配方一致。本地持久候选放在忽略目录 bin/fastnet-history-optional-candidate-20261110，设备已有本地测试源同步更新。包数据库没有执行安装升级。

源码提交：FastNet 33f23cd；OpenWrt 插件 ffbae19。Quickstart 验收与部署改动包含本目录。
