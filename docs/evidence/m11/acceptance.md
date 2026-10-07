# M11 名称与配置写入安全验收

> 验收日期：2026-09-24  
> 测试机：`root@192.168.9.215`  
> 回滚目录：`/root/quickstart-backups/m11-20260924100455`

## 自动测试

- 后端全量：1,302 项通过；
- 静态租约、分类定向 race：106 项通过；
- 前端领域测试：25 项通过；
- `vue-tsc --noEmit`：通过；
- Vite 生产构建与 zip：通过；
- 运维脚本契约、测试机 preflight、amd64 构建：通过；
- `git diff --check`：通过。

## 真机输入边界

以下请求均由 legacy 静态租约 API 发出，因此也验证了经典入口经过共享后端 Validator：

| 输入 | 结果 |
| --- | --- |
| `客厅电视` | 拒绝，返回 DHCP hostname 字段错误 |
| `bad'name` | 拒绝，返回 DHCP hostname 字段错误 |
| `router.local` | 拒绝，返回 DHCP hostname 字段错误 |
| `living-room-tv` | 接受，生成命名 UCI host section |
| 删除时携带旧中文 hostname/非法旧 tag | 接受删除，只使用已经验证的 MAC 定位旧规则 |

三次非法请求前后 `/etc/config/dhcp` SHA-256 均为：

```text
e158cc2078e1b95e68626df491f8d3d0ef2d758bcdeda7992de9b7dcd7acadf4
```

合法请求生成：

```text
dhcp.quickstart_020000110001.name='living-room-tv'
dhcp.quickstart_020000110001.mac='02:00:00:11:00:01'
```

写入与删除后 dnsmasq 均处于 running。测试结束已用验收前快照恢复 DHCP 文件，最终哈希与验收前完全一致。

## 分类与前端

- 后端分类 fixture 已将纯 `ASUSTeK COMPUTER INC.` 厂商证据改为 `ASUS / network / manufacturer_default`；
- 旧后端响应的前端兼容适配器同步为 network；
- 当前测试机 ASUS 设备显示为 network；它已有人工覆盖，因此线上证据来源保持 `manual`，没有擅自清除用户设置；
- 新设备详情不再执行 `displayName → DHCP hostname`；
- 三个入口统一标注“DHCP 主机名（可选）”，带 63 字节限制和 ASCII LDH 提示；
- Playwright 真机复验覆盖 1440×900、1024×768、390×844；三档页面均无横向溢出；中文 Hostname 令保存按钮禁用，`living-room-tv` 恢复可保存；截图及可重复脚本保存在本目录。

## 配置安全

- 静态租约写入不再构造 `uci set ... '%s'` Shell 命令；
- 旧 DHCP 标签入口只接受 `add/modify/delete`、安全 section ID 及 option 3/6 的 IPv4 值；删除旧规则时只使用安全 ID，不再信任旧标题和 option；
- 旧浮动网关入口对职责、IPv4、HTTP(S) URL 和 1～30 秒超时执行权威校验，无法把引号、换行或命令片段带入历史命令适配器；
- 旧限速规则已改为 Go UCI Tree 类型化写入，备注中的控制字符会被清理且不会进入 Shell；
- Go UCI Tree 直接写类型化 section/option；
- dnsmasq 临时文本预检在快照和写入前执行；
- live 写入后同步 reload 并检查进程；
- mutate/reload 失败测试均验证恢复快照。
