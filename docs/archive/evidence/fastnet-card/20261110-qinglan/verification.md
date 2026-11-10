# FastNet 晴岚前端验收（2026-11-10）

结果：当前候选通过本轮验收，已保留部署到 `root@192.168.30.7`。

提交：此前完整工作已先分别提交为 Quickstart `655602c`、FastNet `e8f0480`、插件 `37ab170`；本轮 FastNet `2206ce1`、插件 `b81a2e7`，宿主改动与本记录位于当前 Quickstart 提交。

入口：<http://192.168.30.7:10000/cgi-bin/luci/admin/quickstart>，HTTPS 入口为 <https://192.168.30.7/cgi-bin/luci/admin/quickstart>。当前部署 `fastnet-square-20261110T022144Z`；设备回滚脚本 `/root/quickstart-backups/fastnet-square-20261110T022144Z/rollback.sh`。本次为源码测试候选，未推送或发布正式版本。

## 交付与范围

- 外网测速卡片复用首页原有主操作与更多菜单，去掉测速外跳箭头；比较实际控件的高度、边框、颜色、字体均通过。旧宿主保留回退。
- FastNet 自有十个页面使用晴岚公共 CSS 与令牌，支持独立浅色、深色、跟随系统；嵌入页跟随首页，切换不重挂表单或测速。顶部显示运行核心实际版本 `v0.8.2`。
- 上游设计源固定于 `linkease-desktop` 提交 `7c8e968adf4010b861f41eebf8ff93ef00a44e1b`，完整闭包 50 文件逐项校验；React 文件保留溯源，不进入运行时。
- LAN 测速库与此前已提交代码比较，仅增加可注入传输函数及一次分发调用。Ping、计时、流、并发、上传回退及速率计算未变；适配 `/apps/fastnet` 前缀。
- FastNet 自身 LuCI 增加认证的只读来源能力接口，从 uHTTPd REMOTE_ADDR 读取实际客户端，避免代理 loopback 误判；不信任转发头。核心只增加只读版本接口。
- Quickstart 修改限于卡片宿主公开原有控件、卡片重复 CSS 清理、设计预览和验收脚本。未改 iStore、桌面或应用网关源代码；真实设备首页布局、网络/FastNet 配置、包记录、Quickstart 后端八个受保护路径前后 hash 相同。

## 验证结果

| 验证 | 结果 | 证据 |
| --- | --- | --- |
| 完整前端页面与状态 | 84 项通过：10 页 × 浅深 × 1440/1024/390/320，加主题/草稿/弹窗/错误/存储场景 | [报告](full-ui-final/browser-report.json) |
| 补充交互与本地原生 LAN 压测停止 | 5 项通过 | [报告](interactions-final/browser-report.json) |
| 正式卡片设计预览 | 31 项通过，含两种形态、主题、子窗口结果同步、关闭、失败重试 | [报告](card-preview/browser-report.json) |
| 真机完整流程 | 5 项通过，含统一操作区、版本/来源、换主题保留参数、实际 LAN 完整测量、鼠标停止压测 | [报告](live-complete/browser-report.json) |
| 真机键盘停止 | 4 项通过 | [报告](live-stress-keyboard/browser-report.json) |
| 真机 HTTPS 与手机 | 4 项通过，含主题和移动弹窗 | [报告](live-https/browser-report.json) |
| Quickstart 单元与类型 | 193 项通过、0 跳过；vue-tsc 通过 | 本轮本地执行 |
| FastNet 前端业务 | 27 项通过；LAN 边界 9 项通过；原库归一化逐字比较通过 | [源码边界](source-boundaries.json) |
| Go 核心 | `go test ./cmd/fastnet -count=1` 通过，含版本方法/认证 | 本轮本地执行 |
| LuCI 与包声明 | 12 项来源、10 项嵌入控制器、3 项包契约通过 | `controller-tests.log` 与本轮本地执行 |

真机完整测量实际记录六次 Ping、下载和上传流（当次显示 12.2 ms / 925.9 Mbps / 2.74 Gbps），期间切换主题；该数值仅为本次测试环境结果。压测分别通过实际鼠标和键盘停止，显示“已停止”，不误报完成。页面矩阵中的 WAN/第三方服务响应使用测试数据；不以此冒称外部服务端到端实测，也未重跑历史全部 FN01–FN12。

`full-ui-r*`、`live-LAN*`、`live-stress-final` 是优化过程记录；最终结论以表中最终报告为准。

## 资源与候选

宿主首屏 gzip 增量 20,442 字节（门槛 20 KiB）；卡片入口 8,052 字节（门槛 8 KiB），弹窗胶水按需 2,426 字节。完整前端 JS 67,373 / CSS 11,062 字节 gzip，仅点击打开后加载。详见 [资源报告](resources.json)。

本地交付目录 `bin/fastnet-qinglan-candidate-20261110/` 保存五个 IPK、三架构核心归档、Quickstart Web、输入溯源和校验和，不纳入 Git。核心归档 SHA256 为 `fcb6dbcd41a3c8bdcc3bdd484f8e4756fcd800a75a11f31a84e69a57a233f998`，与插件配方一致。

源码输入见 [source-inputs.json](source-inputs.json)、[完整前端输入](frontend-source.json)；正式设备文件与受保护路径见 [部署记录](deployment-final/deployment.json)。截图保存在对应验收目录；开发容器设计预览仍为 `http://127.0.0.1:5180/`。
