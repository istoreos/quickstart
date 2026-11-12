# FastNet 弹窗单行标题验收

2026-11-10，使用 `scripts/ui-design.sh` 本地预览与 `root@192.168.30.7` 保留的正式 Quickstart 页面验收。

目标：标题栏只显示“外网测速”，移除 QUICKSTART 副标题及 FASTNET · WAN / LAN / NAT / IPv6 工具栏；“新窗口打开”移入标题栏右侧，紧邻关闭按钮。iframe 内部仍是原有完整 FastNet 页面。

- 本地和真实设备：1440 / 390 / 320px，浅色与深色均通过；头部 61px，iframe 距弹窗顶部 62px，无横向溢出。
- 新窗口打开、子页面焦点边界消息、Escape 关闭与恢复卡片按钮焦点均通过。普通“查看上次结果”窄弹窗保持原有头部。
- Quickstart：生产构建、vue-tsc、194 个现有测试通过（默认跳过的生产 chunk 检查单独带 CARD_HOST_BUILD 执行通过）。FastNet 卡片 11 个测试、包契约 3 个测试通过。
- 仅修改宿主弹窗操作接口、FastNet 卡片头部及对应静态资产/版本；没有改变测速、历史、LAN 库或完整 FastNet 前端。
- 宿主 JS/CSS gzip9 相对上一版增加 117B；懒加载 WebDialog 减少 269B。卡片主入口 gzip9 8151B。没有增加依赖。
- 测试部署使用 `fastnet-square-iframe-deploy.py --static-only`，更新 Quickstart 静态资源/模板、卡片静态资源/注册信息及已有本地测试源。没有安装软件包、重启服务或运行带宽测速。
- FastNet PID 保持 1258；核心、历史文件、网络/DHCP/防火墙/FastNet 配置、首页布局/启动器、opkg 记录及 Quickstart 后端哈希一致。详细文件哈希见 deployment.json。
- 测试地址：http://192.168.30.7:10000/cgi-bin/luci/admin/quickstart 。资源版本 fastnet-header-20261110T051707Z。
- 设备私有回滚脚本：/root/quickstart-backups/fastnet-header-20261110T051707Z/rollback.sh（仅静态回滚）。
- 发布候选：LuCI 0.8.3-r2、meta 0.8.3-12、卡片注册 0.8.3-2；FastNet 运行核心仍为 0.8.3，二进制归档哈希保持 8c4eada970617fe607cf9446fbd32f5ffd87678d3da6211c3b22e2566af2d4ee。

FastNet 源码提交：c436e65；OpenWrt 插件提交：5121ea5。Quickstart 提交包含本验收目录。
