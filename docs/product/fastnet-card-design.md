# 首页「外网测速」卡片

FastNet 是 Quickstart 首页的独立可选模块。Quickstart 仅内置名称、介绍、图标、安装包和展示形态等轻量目录；已安装的 luci-app-fastnet 注册带内容哈希的外部组件资源。卡片业务由 FastNet 维护，宿主提供共享 Vue 运行时、公共 CSS、布局、主题和弹层。

默认系统未安装 FastNet。用户从「自定义首页 → 添加模块」选择「外网测速」，先确认安装插件；安装在后台继续，关闭或刷新页面不会重复安装。安装完成后选择放置位置并保存布局；安装本身不自动放置或测速。取消放置不会卸载插件。插件被卸载后保留已保存的位置和显示偏好，显示安装提示。

## 内容与操作

标准与摘要两种形态都显示上次下载、上传速率（Mbps）、上次测速时间、NAT 类型和 IPv6 状态。NAT、IPv6 各自保留检测时间；缺值显示「— / 未检测」，有效的不可用结论与检测错误分别表示。部分失败保留最近有效结果，下载与上传采用同一次任务的完整结果，避免混合两次速率。

卡片直接提供测速、停止和失败重试。运行中保留旧值并显示阶段；移动、切换形态、切主题、关闭页面不会启动新任务。页面返回后接续后台任务。服务或设备重启后原任务中断，不自动重新测速。服务未启用时提供设置入口，由用户启用。

更多菜单提供详情、重新读取状态、插件设置和完整 FastNet Web。详情说明默认外网出口、各项检测时间和带宽影响。范围仅为外网，不包含 LAN 测速、多 WAN 选择、历史图表或完整工作台。

## 颜色与加载

右上角浅色、深色、跟随系统使用统一主题服务。偏好保存在浏览器，跟随系统时实时响应 OS 颜色；卡片及持续打开的安装、详情、错误弹层同步更新公共语义变量。插件不修改 body、不保存独立主题，也不重复监听系统主题。

未添加的模块不下载业务 JS、不读取 FastNet API。已添加卡片进入可视区时动态导入，同一内容哈希资源在页面内复用；Vue 由宿主共享。闲置卡片首次读取后不周期请求，运行中每次查询结束后间隔至少两秒，隐藏后停止前端查询，后台测速继续。加载或 API 故障只降级该卡片。

协议见 [卡片契约](../fastnet-card-contract.md)，实现与分阶段证据见 [实施里程碑](../archive/product/fastnet-card-implementation-milestones.md)。业务组件位于 FastNet 仓库的 `quickstart-card/`，LuCI 接口和注册资源位于插件仓库的 `apps/fastnet/luci-app-fastnet/`，宿主加载器位于本仓库 `web/src/home/cards/`。

## 安装弹窗与改动范围复核（2026-11-10）

安装窗口现在分为插件标识、安装说明和底部操作区。安装/升级为主操作，返回为次操作，重新读取和软件中心作为辅助入口。手机上操作区分行，触控按钮至少 44px。检查、安装中、失败、未知和待升级有对应状态；Esc 关闭后焦点回到模块入口。持续打开的弹层随系统主题实时变化。

该窗口由正式 `HomeLayout.vue` 使用 `cards/FastnetInstallDialog.vue`，预览复用同一组件。窗口代码按需下载，失败提供返回入口；此次宿主首屏 gzip 增量为 20,265 字节，仍低于原 20 KiB 门槛。

范围复核已撤销先前的全局 App.vue 焦点重置改动，将修复限定在外部卡片；安装跟踪和严格包状态检查只用于 FastNet，其他插件保留原行为。原 LAN/设备管理、NAS、Docker、下载和远程服务业务目录未修改，旧设计预览路由也保留。共享 Vue ABI、发现接口、layout v4 与主题变量属于支持外部卡片所必需的公共改动。

本轮通过 193 项前端测试、后端相关套件、26 组弹窗预览检查、82 组卡片回归，以及真实 LuCI 首页的 19 组弹窗/安装检查。旧 FN01–FN12 是前一候选的完整验收，本轮记录验证其后的 UI 和范围修正，不表示重新执行了全部 76 项。

[正式首页弹窗效果](../archive/evidence/fastnet-card/20261110-popup-polish/formal-x86_64-r1/light-1440-zh.png) · [手机深色](../archive/evidence/fastnet-card/20261110-popup-polish/preview-r1/dark-390-zh.png) · [本轮验证与范围复核](../archive/evidence/fastnet-card/20261110-popup-polish/report.json) · [部署状态](../archive/evidence/fastnet-card/20261110-popup-polish/deployment-status.json)。

开发容器 `5180` 提供设计预览。此前两台实验设备的临时验收曾还原；随后按用户要求，已在 `192.168.30.7:10000` 保留部署新版 Quickstart，并通过实际首页确认「添加模块 → 外网测速」与安装弹窗。该机使用 `/etc/opkg/quickstart-fastnet-test.conf` 指向本机候选包源，沿标准 is-opkg 安装流程；当次部署未安装 FastNet；之后用户已在该机安装插件。现有首页 JSON、网络配置和旧手动核心文件未改。`192.168.30.93:80` 仍保持原状态。[当前保留部署记录](../archive/evidence/fastnet-card/20261110-test-deployment/deployment.json)。

## 设计预览

```sh
./scripts/ui-design.sh start
OUTPUT_DIR=/tmp/fastnet-card-review NODE_PATH=/tmp/quickstart-playwright/node_modules \
  node docs/product/ui-design/fastnet-card-review.cjs
```

打开 <http://127.0.0.1:5180/>，默认展示未安装流程。右下「卡片预览」可切换未安装、已有结果、空态、旧值、运行、失败、部分可用、服务关闭、权限、查询失败及安装失败等场景。场景切换保留当前主题偏好。

直接访问 `/?external=present&scene=ready` 查看已安装卡片；加 `&form=summary` 查看摘要，`&tools=0` 隐藏预览工具。预览调用正式首页布局、主题、加载器和独立构建的 FastNet 组件，只替换服务数据及安装反馈，不执行真实安装或测速。预览布局保存在本次页面内存。

历史直接测速交互由 `scripts/ops/fastnet-card-review.cjs` 记录；当前入口 `scripts/ops/fastnet-square-iframe-review.cjs` 覆盖尺寸、两种形态、主题、状态、iframe 测速和结果回写，安装仍由 `fastnet-install-dialog-review.cjs` 验证；旧直接测速的故障记录由 `fastnet-fault-review.cjs` 保留，当前 iframe 超时/重试覆盖在新入口中。正式设备另用 `fastnet-live-*.cjs` 验证，不以预览模拟替代。

早期原型截图保留在 `docs/product/ui-design/fastnet/v2/`，只作历史设计追溯。

## 方形卡片与完整 FastNet 子窗口（2026-11-10）

标准卡片以下载主速率、上传辅速率为视觉中心，使用随卡片高度展开的浅色/深色面板；NAT 和 IPv6 各有状态块，保留上次测速时间。短卡片自动收紧间距，摘要形态继续使用紧凑排版。没有真实速率历史，不绘制虚构趋势或进度表盘。状态圆点同时配有文字，不把 NAT 类型直接评定为网络好坏。

点击「测速」「查看进度」或菜单中的「完整 FastNet」打开首页内的宽弹窗。完整页面及其 Vue 只在点击后加载，关闭后移除 iframe。打开不自动测速；子窗口默认的外网体检/外网测速工作区使用与首页相同的持久任务接口，点击开始才提交一次任务。关闭不会停止已提交任务，首页恢复观察同一个任务并更新结果；停止测速可在子窗口或首页进行。原 FastNet 其他页面仍可从完整页面菜单访问。

宿主通过向后兼容的 `openDetails(component, props, {wide, onClose})` 提供宽弹层和关闭通知。打开 iframe 时首页暂停查询，避免双轮询；关闭后立即读一次摘要。框内 Escape 返回首页并恢复焦点，Tab 边界与外层弹窗衔接。主题仅由 Quickstart 决定，向 iframe 发送有效 light/dark 消息；消息校验同源及发送窗口，页面原地更新，不重载或重复测速。浅色嵌入页沿用 FastNet 现有浅色布局，深色沿用其深色布局，完整页面独立打开的行为保持原样。

新增的认证 LuCI `/admin/services/fastnet/embed` 只接受 root 的 GET，并选择同源 `/apps/fastnet/` 入口。FastNet 自己的插件声明使用现有 app-base 代理能力，核心首页跳转与前端 API 使用相对/带前缀路径，支持 HTTPS 同源访问。没有网关时显示加载失败、重试和新窗口打开，不隐式跳转 HTTP 端口。没有修改 iStore、LinkEase 桌面、应用网关源代码，也不写网络设置。加载握手超过 15 秒移除 iframe；失败可明确重试。

卡片主体 gzip 8,042 字节，弹窗胶水另一个按需 chunk 1,859 字节；宿主首屏相对原基线增量 20,425 字节，分别满足原 8 KiB / 20 KiB 门槛。完整 FastNet 前端不进入首页初始资源。`scripts/ui-design.sh` 预览使用正式完整前端，仅将 iframe API 接到父页面的模拟服务，不转发到路由器。

当前验证与测试机部署以 [方形卡片/iframe 验收记录](../archive/evidence/fastnet-card/20261110-square-iframe/report.json) 为准；历史 FN01–FN12 及旧弹窗记录保留用于追溯，不表示对当前候选重新执行了全套验收。


## 晴岚完整前端与统一卡片操作区（2026-11-10）

FastNet 卡片右上角现在直接复用 Quickstart 原有主操作和更多菜单，使用相同的高度、边框、分隔线、颜色与字体，去掉外跳箭头；保留旧宿主的原生回退。无需在宿主内复制 FastNet 的业务界面。

完整 FastNet 前端使用 LinkEase 晴岚 V1.0：同步上游 manifest 的完整50文件闭包并逐项验hash，Vue只使用原始公共CSS和令牌，不引入React运行时。独立页面支持浅色/深色/跟随系统，iframe由首页决定有效主题；主题只改外观，表单、LAN参数和任务不重挂。顶部通过只读API显示实际核心v0.8.2。

自有10个页面、导航、结果、表单、弹窗、日志与失败反馈均迁移；外部NAT/IPv6服务文档保持原内容。LAN库仅增加传输注入：计时、六次Ping、并发、流、服务端上传计时和速率公式保持原样。代理模式由FastNet自身LuCI来源接口读取真实uHTTPd客户端，不把网关loopback当作浏览器，也不改网关或iStore。

[本轮验证记录](../archive/evidence/fastnet-card/20261110-qinglan/verification.md)包含最终84组全页/状态检查、卡片回归、实际LAN流与压测停止、HTTP/HTTPS和范围核对；源码已先提交此前全部工作，再单独提交此次改造。测试机仍为 `192.168.30.7:10000`，并保留回滚副本。
