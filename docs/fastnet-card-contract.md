# Quickstart 外部卡片协议 v1

状态：FN01 冻结。JSON 样例见 [fastnet-card.json](contracts/fastnet-card.json)，实施与验收按[里程碑计划](archive/product/fastnet-card-implementation-milestones.md)执行。

## 所有权与兼容

Quickstart 内置可安装目录和公共外壳；`luci-app-fastnet` 安装 `/usr/share/quickstart/cards.d/fastnet.json`，资源位于 `/www/luci-static/fastnet/quickstart/`。FastNet 核心单独安装不代表卡片已安装。不修改 iStore。目录条目不带可执行地址，布局只保存 ID、位置和展示偏好。

注册协议 `schemaVersion=1`、ABI `quickstart-card-v1`；插件导出 `apiVersion=1` 与默认 Vue 组件，只接收 `context` 属性。宿主与插件编译器均固定 Vue **3.5.44**，使用 `/luci-static/quickstart/card-runtime-v1.js` 的 Vue 导出，禁止打入第二份 Vue。运行时提供响应式只读的 `theme` 与 `form`、同源请求、语言和宿主详情弹层。插件不访问宿主私有 store，不挂载第二个应用。

静态入口只允许 `/luci-static/<plugin>/.../<name>.<content-hash>.js`，不含 query、fragment、转义、路径回退或其他源。注册最多 32 项，单文件最多 8 KiB、聚合最多 64 KiB。重复 ID 的所有条目拒绝，内置 ID 不得覆盖；未知协议/ABI/字段拒绝并报告，不执行探测脚本。manifest 不能作为安全沙箱；仅系统包安装的同源代码有执行权限。

兼容矩阵在 JSON 中；升级中的旧哈希 404 最多触发一次重新发现，随后明确重试。失败 Promise 移出宿主加载缓存。禁止用时间戳绕过缓存。显式重试最多使用三次固定 `?qs-card-retry=N` 地址，以避开浏览器对失败 ESM 的缓存；成功的版本继续复用 Promise。旧插件缺少注册时提示升级/修复，不在首页初始加载完整 Web；显式点击可以在弹窗中按需加载完整页面。

## 数据与时间

卡片仅检测默认 WAN 路由。速度值使用 **bit/s**，展示为十进制 Mbps，未知为 `null`，不可用不能转成 0。每项 `status` 为 `not_tested / ok / unavailable / error`；`unavailable` 是一次有效的不可用结论，`error` 是没有得到结论。未检测无时间、值或错误；成功项有独立 UTC RFC3339 时间；探测错误有错误码，不覆盖上一有效值。

`latest.measuredAt` 仅在同一次任务下载、上传均有效时更新，是“上次测速时间”。NAT、IPv6 分别保留实际完成时间；混合时间的结果明确显示各项时间与来源，不能把读取时间当检测时间。`latest.runId` 指最近完整带宽结果来源。单项成功可更新该项，最近完整带宽结果保存在可选 `latest.bandwidth` 中；卡片主速率与主时间同时使用这一完整快照，单项新结果放在详情中。每项可选 `runId` 标识来源，`lastTask.issues` 标识本次探测失败而旧值仍保留的项目。24 小时前的结果标为旧结果；这是显示提示，不自动测速。

结果可能部分成功；错误和空态保留上次有效数据。`summary` 是只读快照，禁止启动探测、安装或启用服务。没有结果时 `latest=null`，无活动任务时 `task=null`。

## LuCI 接口与任务

公开基址 `/cgi-bin/luci/admin/services/fastnet/card`，响应 JSON 且 `Cache-Control: no-store`：

| 方法 | 子路径 | 输入与结果 |
| --- | --- | --- |
| GET | `/summary` | 服务状态、实例 ID、最近结果与活动任务 |
| POST | `/run` | `{"idempotencyKey":"<16-64 ASCII chars>"}`；202 返回 task，相同键返回同一 task |
| GET | `/task?id=<taskId>` | 返回同一实例的任务；找不到为 404，不创建任务 |
| POST | `/cancel` | `{"id":"<taskId>"}`；幂等取消，返回当前 task |

写操作必须经过 LuCI 登录与 CSRF 校验；拒绝 GET mutation、畸形/超大输入。公开接口不能返回核心 token、进程命令或堆栈。401/403 权限、404 缺项、409 busy、422 参数、503 停用/服务不可用、502 上游协议错误分别保留。浏览器不直连端口 3200；controller 使用 root 所有、0600 的 `/var/run/fastnet-card.sock`，Go 校验连接对端 UID；与外部监听端口和核心 token 无关。宿主使用现有 `window.token` 的 LuCI 写保护值，兼容自定义头及标准 token 查询参数，不将它转发给 Go。controller 只向受约束的本机服务地址请求，在 8 秒内返回，不执行长任务命令。使用实际服务监听信息，不能盲信未应用的 UCI 端口。

Go 核心持有任务，寿命不绑定 HTTP 请求。状态转移见 JSON，终态不可复活；服务启动读取未完成任务并标 `interrupted`。设备同时最多一个占用带宽的任务，原 Web 的快速/多流/Speedtest/Homebox 客户端入口共享门闩。不同幂等键在 busy 时返回当前任务与 409；相同键即使响应丢失也不二次执行。服务实例 ID 防止旧实例任务混淆。幂等重放覆盖保留的最近 32 条任务，调用者每次明确新测速生成新键。单任务截止 120 秒，取消/截止后 2 秒内释放网络资源；取消后的迟到结果不能写成功。

内存保留最多 32 条任务摘要，每任务日志不超过 64 KiB；进度仅内存更新。`summary.lastTask` 保留最近终态和分项错误；`service.persistenceError` 明确报告保存失败。这些都是 v1 的可选附加字段，不改变必需字段或注册 ABI。创建、完成、取消/中断等状态转换才原子落盘；落盘失败保留上次有效文件并报告持久化错误。重启不自动测速，卸载卡片/离开页面只停止前端查询。

## 布局与安装

layout v4 保持 v3 的 rows/widgetPreferences 结构，新增符合 `plugin.<id>` 的外部 ID，偏好形式只允许 auto/standard/summary。前后端一致校验，最多 32 个外部 ID、每个单实例。已卸载项允许存留；缺注册不是布局损坏，不丢弃位置或偏好。v3 无损迁移，launcher schema 不变，原冲突、备份、future 只读保护继续生效。旧客户端遇 v4 必须只读，不降级覆盖。

未安装时点“添加”先显示安装说明，明确用户确认后安装 app-meta-fastnet。安装失败/取消不改布局；安装完成重新发现后再添加。插件安装独立于布局草稿：放弃草稿不自动卸载。已有布局中的缺失插件显示可修复占位。服务停用须用户前往设置处理，不自动启用。

## 主题与资源

主题只由宿主 `theme.ts` 管理，偏好 system/light/dark，有效 light/dark。system 对应 LuCI body[theme=normal]；插件接收只读响应式快照，不自建媒体监听、不写 body/localStorage/UCI。主题切换不得重新导入、挂载、读取摘要或重测。迟到懒加载的首帧使用当前主题。

公共变量 `--qs-card-panel/text/muted/line/soft/accent/good/warning/error/focus/disabled`，所有宿主弹层共享且持续更新，禁止打开时复制颜色快照。图标 currentColor、状态带文字、按钮支持键盘和焦点。

未添加/未安装不加载业务 JS；可视区懒加载，成功版本每页只导入一次。闲置首次读取后无周期轮询；运行期完成上次请求后至少 2 秒再查，8 秒截止，连续 3 次失败停止自动查询。隐藏取消查询而不取消后台任务，恢复先读原任务。完整预算 B01–B10 与主题矩阵 TH01–TH12 以里程碑计划为准，不在验收失败后放宽。


## 可选的宽详情弹窗与 FastNet 嵌入

`context.openDetails(component, props, options?)` 可选第三个参数 `{wide?:boolean, headerAction?:{href:string,label:string}, onClose?:()=>void}`。省略时保留原 560px 详情弹窗；wide 使用受视口限制的宽弹窗和单行标题，不显示 QUICKSTART 副标题。可选 headerAction 将同源“新窗口打开”链接放在标题栏关闭按钮之前；业务组件不再重复渲染工具栏。关闭回调最多一次，宿主卸载时只清理，不触发已卸载业务。详情组件可发出 `close` 事件。iframe 为键盘焦点节点，子窗口通过已校验的同源消息处理 Escape 与 Tab 边界。

FastNet 卡片入口仍是内容哈希 ESM，允许从同一插件目录按需导入哈希弹窗 chunk；构建 provenance 列出全部 JS 资源。入口共享宿主 Vue；完整 FastNet 的 Vue 只在显式打开 iframe 后出现，关闭即移除 iframe。iframe 消息仅有 ready、close、focus 和有效主题，不传 token 或测速数据；结果通过原持久任务接口读取。

`GET /cgi-bin/luci/admin/services/fastnet/embed?theme=light|dark` 为认证 root 的导航入口，使用现有 `/apps/fastnet/` 同源网关。插件自身声明 `backend.proxyMode=app-base`、`pathMode=strip-public-base`；FastNet 的首页重定向及前端 API 尊重公共前缀。无网关时不降级到混合内容的外部 HTTP iframe。子窗口中的外网工作区复用原卡片模型与 `/api/card/*` 接口，不调用旧 SSE 测速后再尝试拼接摘要。


## 可选的宿主标准操作控件

`context.components.PrimaryAction` 和 `Menu` 是可选的宿主 Vue 控件，使用同一共享 Vue SDK。PrimaryAction 接收 icon、默认文字 slot 和 click；Menu 接收 label、`{id,label,disabled?}[]` items，并发出 select(id)。它们直接复用首页已有的控件、边框、分隔线、焦点与键盘行为，不读取插件业务状态。旧宿主没有此字段时，插件使用原生回退。组件注册不进入安装目录元数据，测速与数据仍由 FastNet 负责。

FastNet 完整前端按晴岚公共闭包独立构建；完整工作台不会进入 Quickstart 首屏。`GET /api/version` 显示实际核心版本。经应用代理运行的页面使用认证的 LuCI `GET /admin/services/fastnet/browser-origin` 判断真实浏览器来源；该接口只返回布尔能力，不信任浏览器转发头、不触发测速。LAN 的公开路径由原门面适配，原测量逻辑、计时、流与公式不变。
# FastNet 0.8.3 统一历史补充

FastNet 自有完整页面在独立端口和 iframe 中采用同一业务。卡片仍使用 `quickstart-card-v1`，summary schema 保持1；新增可选 `latestResults`（各指标含自身runId/时间/来源）、`revision`、`historyCount`。旧 `latest` 为兼容投影；LAN从 `lanPing/lanDownload/lanUpload` 读取。完整历史由FastNet核心 `/api/history/*` 管理，最多32条；Quickstart不存第二份测速记录。

卡片在可见且前台时每15秒读取空闲摘要，任务期间每2秒；离屏、隐藏或打开iframe时暂停，恢复可见/关窗立即重读。原“空闲只读取一次”预算由此需求替代。历史页面及结果详情按需加载；卡片入口仍检查8KiB gzip门槛。

`/admin/services/fastnet/embed?view=history` 可进入公共历史页面，普通打开仍默认快速体验。`fastnet:results-changed` 仅为失效通知，不接收跨窗口结果正文。核心已接受的任务独立于窗口；浏览器LAN/NAT/IPv6需要原浏览器继续执行。LuCI停止服务回退读取支持磁盘schema1/2。
