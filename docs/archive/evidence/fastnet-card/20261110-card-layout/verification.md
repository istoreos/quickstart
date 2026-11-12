# FastNet 首页卡片：WAN / LAN / 网络状态分层

2026-11-10。目标：WAN 放上方，LAN 放中间，NAT / IPv6 放最下面；保留现有测速、历史和浅色/深色/跟随系统行为。

## 任务树与结果

- 结果布局：直接调整 FastNet quickstart-card 的模板顺序；LAN 改为与 WAN 对齐的下载/上传两列，保留自己的最近测速时间；NAT / IPv6 收紧为底部状态区。完成。
- 视觉：沿用宿主语义颜色，移除固定色值渐变；强调 WAN 下载，LAN 数值次一级，时间和单位弱化。字号依据卡片宽度，紧凑形态依据实际高度。完成。
- 小尺寸：普通 280×260、摘要 240×148、触屏运行状态及触屏摘要均检查。紧凑触屏 LAN 使用一行，保留 44px 操作目标；WAN 最小高度避免数据溢出；较矮卡片提前使用紧凑布局。完成。
- 交互：测速仍懒加载默认 FastNet 快速体验 iframe，打开弹窗不自动测速；关闭后不改变历史或最新结果。完成。
- 构建与交付：通过 SSD 流程构建独立卡片，重生成 LuCI 自有资源、注册及 provenance。卡片版本 0.8.7-2、LuCI 0.8.7-r2、meta release 17；FastNet 核心版本、归档哈希不变。完成。

## 验收

- 通过 scripts/ui-design.sh 构建并运行正式外部卡片与完整 FastNet 前端。
- 18 项浏览器检查通过：浅/深/跟随系统，576×405、320×330、280×260、280×148、240×148，以及空数据、测速中、部分失败、权限失败、触屏和默认弹窗入口。无页面异常，没有启动测速。检查同时断言真实卡片高度、横纵向无溢出、WAN 数据处于其面板内、WAN → LAN → NAT / IPv6 的几何顺序。
- 卡片 model 原有 11 项测试通过；OpenWrt 包契约 3 项通过；三仓库 git diff --check 通过。
- 实机浅/深色截图和入口检查通过，历史摘要前后一致，记录仍为 32 条。

## 部署及边界

测试机 root@192.168.30.7；首页 http://192.168.30.7:10000/cgi-bin/luci/admin/quickstart 。

仅部署 FastNet 外部卡片 JS、provenance 及两处注册信息；旧哈希资源保留供已有页面使用。当前入口 `/luci-static/fastnet/quickstart/card.182c88e6.js`，SHA256 `4c790601633787b23a2171614c99dbeb18a81679c4d52584dd3dc0983846af85`。卡片 gzip 约 8.14 kB；没有新增运行依赖或请求。

回滚：`/root/quickstart-backups/fastnet-card-layout-20261110T120321Z/rollback.sh`。

部署前后 FastNet/Quickstart 二进制、进程 PID、历史文件、布局、网络与 FastNet 配置、包记录、LuCI controller/model 及宿主模板均核对保持不变。未安装 IPK、重启服务、更新软件源、发布目录或触发测速；这是测试机静态资源更新，不是软件源正式发布。

源码业务改动仅 FastNet quickstart-card 的 Card.vue、style.css 及 manifest。OpenWrt 仅对应资源/注册、包版本依赖和包契约；Quickstart 本轮仅新增此验收记录与浏览器检查脚本，未改其产品业务源码。完整部署/回退与实机检查工具、候选静态资源持久保存在忽略目录 bin/fastnet-card-layout-20261110/。验收完成时本轮尚未提交或推送；随后按用户要求作为 0.8.8 提交并发布，见 ../20261110-release-0.8.8/verification.md。
