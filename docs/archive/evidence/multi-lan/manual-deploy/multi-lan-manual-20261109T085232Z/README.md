# 192.168.30.93 多 LAN 人工测试候选

前后端和 NetPolicy ARM64 程序已编译并部署；实际 LuCI 页面及临时真实会话鉴权接口冒烟通过。页面可选择全部局域网、lan、lan1、lan2，没有新增网络配置。

入口：http://192.168.30.93/cgi-bin/luci/admin/quickstart/devicemanagement?network=all

浏览器使用正常 root 登录。生产资源带本次版本标记；如已有旧页面，刷新后进入。lan1/lan2 现无 IP 地址或 DHCP 绑定，处于只读状态；NetPolicy 保持原有关闭配置。需要测试实际地址分配、限速、流量或断网时，应先按测试拓扑完成相应网络配置并启用对应网络的服务。

本次部署仅用于人工验证。M10 第 5 轮的计划断网 IPv4 验收失败仍未修复，最终资源/稳定性验收未通过，不表示已达到上线标准。

## 备份和回退

远端成套备份：/root/quickstart-backups/multi-lan-manual-20261109T085232Z

备份包含部署前的程序、页面、配置及应用数据。以下命令会恢复部署前状态，包括配置和数据；人工测试后新增的设置会被回退。它只重启 Quickstart/NetPolicy，不执行 network/dnsmasq/firewall 的服务重启：

    sh /root/quickstart-backups/multi-lan-manual-20261109T085232Z/rollback.sh /root/quickstart-backups/multi-lan-manual-20261109T085232Z multi-lan-manual-20261109T085232Z

完整状态和产物哈希见 deployment.json；实际页面冒烟见 browser.json 和 devices-all.png，网络清单及设备作用域读回见 smoke.json。备份内容未写入仓库。
