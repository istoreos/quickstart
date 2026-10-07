# M15 验收记录

> 日期：2026-09-24  
> 测试机：`root@192.168.9.215`  
> 回滚点：`/root/quickstart-backups/m15-20260924191939`

## 结果

- 新增统一 `DeviceNetworkPolicy` 接口，地址预留与上网路线共用一次 UCI 快照、提交、dnsmasq 校验和失败回滚。
- 对外只返回稳定 `targetId`、用户名称和生效状态；响应与页面均不包含 DHCP tag、option 3/6。
- 详情页在三档视口均以“地址与上网路线”呈现，默认、高级信息和错误原因层级清楚，无页面横向溢出。
- 无变化提交返回 `changed=false / active`；联合写入返回 `changed=true / pending_renewal`，文案为“等待设备重新获取地址”。
- 真机联合写入后已恢复 DHCP 快照；最终 `/etc/config/dhcp` SHA-256 为 `e158cc2078e1b95e68626df491f8d3d0ef2d758bcdeda7992de9b7dcd7acadf4`，dnsmasq 正常。

## 自动门禁

- Go 全量测试：通过。
- `go test -race ./service ./modules/lancontrol`：通过。
- 前端 30 项测试、`vue-tsc --noEmit`、生产构建：通过。
- OpenWrt 部署契约与 `git diff --check`：通过。
- 真机 API、联合写入/恢复、1440×900、1024×768、390×844 视觉验收：通过。

## 构建物

- `bin/quickstart.amd64`: `311a46590c5f3075a596b6e116a44e2276b103f9fb9211d8ab8ccd259ea81384`
- `web/dist/quickstart_web.zip`: `bfad2ecc8fa29f531e1887b7e24d95b9d7b23281c960f87ea7a81e17e336ee2e`

M15 结论：达成，无需优化轮次。
