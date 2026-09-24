# Quickstart 0.14.0-r2 发布候选验收

## 候选物

| 候选物 | SHA-256 |
| --- | --- |
| `bin/quickstart-binary-0.14.0.tar.gz` | `86ac0564958d12d59d2583c59975a52a116ce41380e50cc214ba1b1e251f3116` |
| `web/dist/quickstart_web.zip` | `3bd1b69f83088a051fcf27aaede6fd33c0c900236793cdc9f96484324731dc01` |
| `index.js` | `8687b3eebefc5f3a60c5b8dd6e0c558677f137b1bbd16efda2ca4c87f401be48` |
| `style.css` | `ab2291ee796a3e5ab13578fe449a7b8d52efd0a999039552b68010fc643c34b6` |
| `en.json` | `3e4d9a7465a8af6b87670c8c8f3e751ff78b4ae2e3ff827449742ebb5e6993f9` |
| `zh-cn.json` | `799a23b2c098bf689311a3cb54dead69d472234861cc84c1ecb2f116f5bea1f3` |

OpenWrt feed 的后端版本、LuCI 版本和资源缓存版本均为 `0.14.0-r2`；后端源包哈希与候选 tarball 一致。`/etc/quickstart/device-classifications.json` 已加入 conffiles。当前仓库不含 OpenWrt SDK，因此候选物是可供 feed 构建的预构建后端 tarball、LuCI 安装树和一致版本配方，不宣称已经产出 `.ipk`；未执行外部上传或正式发布。

## 自动验证

- 分类样本：60 个，12 个场景组（11 个可选类别 + fallback），20+ 原始厂商变体，10+ 短品牌；全部通过。
- 后端：89 个包、1291 项通过；`service` 与 `lancontrol` race 定向 567 项通过。
- 前端：23 项通过；TypeScript、1008 模块生产构建通过。
- 素材：12/12 原创通用图标存在且浏览器加载成功；无品牌/logo 文件名；来源和统一生成提示词已记录。
- OpenWrt package contract 与 `git diff --check` 通过。

## 测试机灰度

- 地址：`192.168.9.215`；运行二进制报告 `0.14.0`。
- 本地与远端后端、JS、CSS、中英文资源、LuCI 模板哈希一致。
- 升级前创建的持久人工分类在升级和 quickstart 重启后保留；boot-local 覆盖被清除；验收后已恢复自动，存储为空。
- `/etc/config/*` 验收前后清单哈希均为 `373767320f61df45a1c09002cacc1b822b0540698cd6196ab29b0ff4f3087f93`。
- 1440×900、1024×768、390×844：页面与抽屉无横向溢出，11 个类别选择完整，键盘焦点可见，分类图标无破图，控制台新增错误 0。
- 真实清单 26 台设备分类契约全部合法；25 台为 `computer/fallback`，1 台为 ASUS `manufacturer_default`，不存在 fallback 路由器。
- 超长品牌在移动列表省略，详情保留完整品牌和原始厂商。
- 文件级回滚至 `0.13.0` 后旧页面可用、无页面错误；随后恢复 `0.14.0` 并重新通过三档验收。
- 回滚备份：`/root/quickstart-backups/m10-202609240600`。

截图和机器可读结果位于本目录。正式上传预构建源包、构建/发布 `.ipk` 仍是独立外部发布动作。
