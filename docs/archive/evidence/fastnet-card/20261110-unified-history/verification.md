# 统一页面与32条历史：验收记录

结果：H01–H06通过。FastNet提交 `03c5d66`，插件提交 `5af98a7`；Quickstart改动和本记录位于当前提交。本轮FastNet为0.8.3，元包0.8.3-11；代码与候选均为本地交付，未推送或正式发布。任务树在FastNet仓库 `docs/product/unified-history-plan.md`，接口说明在 `docs/product/unified-history.md`。

| 阶段 | 可测试目标 | 结果与证据 |
| --- | --- | --- |
| H01 | 独立、代理、iframe默认快速体验一致，打开不测速，主题不重挂 | [本地完整流程](browser-final/browser-report.json)、[真机补验](live-complete/browser-report.json)通过 |
| H02 | 同一次快速体验一条记录；分类独立更新；32条与字节上限；迁移/重启/失败保留 | 9项新增Go场景与原核心race测试通过；33次后历史32、旧LAN最新值保留；[冷启动](cold-restart.json)通过 |
| H03 | 原生JSON/SSE入口统一归档，互斥、重连去重、关闭继续、取消迟到结果保护 | 原处理函数适配与Go回归通过；[真实快速体验](live-pass/browser-report.json)中的WAN/LAN成功；独立NAT关闭页面后也成功归档 |
| H04 | 原LAN库保留；浏览器结果独立；停止/失败不误报 | 31项前端检查通过（含原LAN、编排、保存重试及HTTP无randomUUID）；[本地真实LAN/停止](browser-final/browser-report.json)通过 |
| H05 | 卡片四类最新值与各自时间；公共历史页；空闲更新；两种主题与手机 | 11项卡片模型检查通过；[真实卡片](live-pass/card-four-latest-dark.png)、[历史/手机补验](live-complete/browser-report.json)通过 |
| H06 | 版本依赖、候选、实际HTTP/HTTPS、停机快照、范围核对 | Quickstart194项、vue-tsc、LuCI12+12+25项及包契约3项通过；[部署](deployment-final/deployment.json)、[停机/重启](cold-restart.json)通过 |

实现修正次数：H01/H02/H03为0，H04为1，H05为1，H06为3。详见 [循环记录](iterations.json)。H06第三次修正将仅安全上下文提供的randomUUID改为getRandomValues，修复局域网HTTP入口；随后产品实现未再修改。

## 真机与过程记录

最终保留部署：<http://192.168.30.7:10000/cgi-bin/luci/admin/quickstart>，HTTPS亦可使用。部署编号 `fastnet-square-20261110T041637Z`；设备私有回滚脚本 `/root/quickstart-backups/fastnet-square-20261110T041637Z/rollback.sh`，包含部署前核心、控制器/模型、静态资源及历史文件。服务与版本已核对。没有OS重启；仅验证FastNet服务冷启动。

真实快速体验完整成功：原生WAN/NAT/IPv6、实际LAN流、浏览器检测合为一条记录；卡片读取同一快照。随后独立NAT曾成功，也遇到一次 `stun timeout`；失败记录只含NAT，上次有效NAT和WAN/LAN保持，卡片显示本次失败。该环境失败属于验收预期分支，不伪造成功结果。

`live-pass`前三项真实流程通过，后续原脚本对异步条件的错误等待导致提前采样；`live-complete`以已保存的真实数据继续验证剩余界面，未重复WAN/LAN流量，最终通过。过程目录 `live/live-final/live-pass/live-finish/live-verified`保留原始失败报告；脚本的Promise等待、结果刷新等待、手机懒加载定位和“每次STUN必须成功”的断言已按真实语义修正。产品修正与脚本问题在循环记录中区分，未删除失败证据。

停机时LuCI仍返回WAN/NAT/IPv6/LAN快照；重启后8条真实/既有记录及最新结果保留，历史文件12511字节、0600，重启前后SHA一致。自检期间误建的一条空诊断会话已按唯一ID清理，没有删除测量或最新结果，见 [清理记录](probe-cleanup.json)。

本地完整流程使用真实核心归档和原LAN端点，原生WAN与第三方回应用明确测试数据，验证32条裁剪及失败/停止；真实WAN证据来自上述`.7`执行。跨架构核心完成编译及候选打包；本轮没有声称其他硬件部署或重跑历史全部FN01–FN12。

## 资源与范围

卡片入口gzip8126字节，小于8KiB；详情1365、弹窗2471字节按需加载。完整前端主JS67997、CSS10539字节gzip，历史JS2405、CSS867字节按需加载。宿主生产前端源码未改，原首屏增量20442字节保持，见 [资源报告](resources.json)。预览脚本已同步统一页面与内存历史，仍只模拟服务，不向路由器转发。

[业务保护](protected-business.json)确认原LAN测量库、原生快速体验/多流/IPv6引擎逐字节不变；修改集中于FastNet Web编排、持久归档、卡片与其LuCI适配。Quickstart源码仅改设计预览与运维/验收；iStore、桌面、网关以及其他首页业务均未修改。部署及验收后网络/DHCP/防火墙/FastNet配置、首页布局/启动器、包记录、Quickstart后端八个路径hash保持原值。

源码溯源见 [输入](source-inputs.json)、[完整前端](frontend-source.json)。候选包、核心归档、Web和SHA256SUMS保存在本地 `bin/fastnet-unified-history-candidate-20261110/`，不纳入Git；核心归档SHA256为 `8c4eada970617fe607cf9446fbd32f5ffd87678d3da6211c3b22e2566af2d4ee`，与配方一致。
