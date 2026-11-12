# FastNet 卡片候选版本安装与回退

本次交付为三个工作树的本地源码候选，不是已发布的软件源版本。FastNet 为 `0.8.2-1`，LuCI 为 `0.8.2-r1`，元包为 `0.8.2-10`；Quickstart 候选包含对应原生后端和首页静态资源。正式发布前通过项目原有 SDK/feed 流水线构建，复核相同源码与依赖，不将本地候选当成已发布 Release。

## 构建与追溯

所有本机构建通过 `/var/cache/build-cache/run.sh -- COMMAND`，输出到其 `BUILD_CACHE_WORK_DIR`。Quickstart 与 FastNet 按各自 `go.mod` 的工具链构建，关闭 CGO，目标 Linux；前端锁定 Vue/compiler `3.5.44` 和 ABI `quickstart-card-v1`。

```sh
# 在 Quickstart 仓库：amd64→x86_64，arm64→aarch64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go -C backend build -trimpath -ldflags=-s -o "$OUT/quickstart.x86_64" ./cmd/backend
cd web && npx vue-tsc --noEmit && npx vite build --outDir "$OUT/web"

# 在 FastNet 仓库：amd64、arm64、arm（GOARM=7）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o "$OUT/FastNet.x86_64" ./cmd/fastnet
bash scripts/build-quickstart-card.sh --package "$APPS/apps/fastnet/luci-app-fastnet"
```

插件的 `tests/build_card_candidates.py` 接收三架构核心、明确提供的 SDK `ipkg-build`、`po2lmo`、`meta.mk`，执行真实 runtime/meta 安装配方并生成五个本地测试 IPK。LuCI 资源在候选中不二次压缩。`candidate.json` 记录源码文件、工具和每个包摘要；`fastnet-source-evidence.py` 记录实际编译的 Go 文件、依赖模块及前端输入。仅 x86_64、aarch64 有本轮设备验收，arm 只提供核心编译产物。

## 安装顺序

1. 在设备上核对架构、可用空间和当前版本。保存原 Quickstart 后端与整个静态目录、LuCI 模板，以及 `/etc/quickstart`、`/etc/config/fastnet`、`/etc/fastnet/card-state.json` 和原插件包。设备配置备份留在私有目录。
2. 校验候选 `SHA256SUMS`。先升级 Quickstart 后端和整套首页资源，保持原 LuCI 入口；为入口 JS/CSS 更新统一资源版本，避免旧入口命中新缓存。勿单独覆盖 SDK 或只换一个入口 chunk。Quickstart 本身不会自动安装 FastNet。
3. 用户在「自定义首页 → 添加模块 → 外网测速」确认安装。正式源就绪后沿现有 iStore 安装 `app-meta-fastnet`；本地离线验收可一次向 `opkg install` 传入该架构的核心与三个 all 包。不得用强制忽略依赖掩盖缺包。
4. 安装完毕，选择位置并保存。服务关闭时先进入插件设置启用，随后手动测速。安装、启用、放置均不自动发起测速。

## 验证

确认 `/usr/share/quickstart/cards.d/fastnet.json` 指向实际存在的哈希 JS；资源直接由 `/luci-static/fastnet/quickstart/` 提供。在已登录 LuCI 中，`/cgi-bin/luci/istore/home/v1/cards/` 可发现该模块，`/cgi-bin/luci/admin/services/fastnet/card/summary` 返回同源摘要。

实测 ARM 的 opkg 在保留旧包、打印依赖不满足时可能返回 0，因此同时核对包版本、依赖诊断和卡片注册，不能仅凭退出码判断升级完成。

首次没有结果应显示缺值；一次真实测速后核对同一任务 ID 的速率、带宽时间及 NAT/IPv6 独立时间。验证停止保留上次结果，刷新继续同一任务，服务/设备重启显示中断并保留历史，不自动重试。切换三种主题、OS 颜色、标准/摘要和弹层，检查结果和任务身份不变。

异常检查包括未登录/会话过期、CSRF、服务停用、旧核心缺少私有 socket、资源缺失、ABI 不兼容、软件源连接失败及最低包版本不满足。无法确认安装状态时先重新读取或前往软件中心；禁止通过重复点击重新启动未知安装。

完整自动化入口、设备报告和资源预算见 [阶段验收计划](archive/product/fastnet-card-implementation-milestones.md) 和 `docs/archive/evidence/fastnet-card/20261109-implementation/`。实机脚本只允许两台明确列出的实验设备，备份和恢复原配置与包记录；设备重启必须启用 `--persistent-reboot`，使用可跨重启保留的私有备份和 `RESTORE.sh`，不能重启普通 `/tmp` fixture。

## 升级、卸载与回退

先升级宿主，再升级核心和 LuCI/元包，最后重新发现。插件原子注册前先确认资源存在。未知 ABI 提示升级宿主，旧插件缺少注册提示升级或修复，均不加载完整 Web 代替卡片。

卸载 LuCI/元包只删除其自有注册与资源，不删除首页布局。外部 ID、位置和形态偏好保留，重新安装后可恢复。停止服务或卸载前确认是否有需要结束的测速任务。

回退插件时恢复匹配的核心、LuCI 和元包，不保留新 manifest 指向已删除资源。回退 Quickstart 时整套恢复后端、静态目录和对应 LuCI 资源版本。旧客户端把 layout v4 视为 future，只读保留；不要把布局强改为 v3，也不要用默认布局覆盖。若确需完全恢复旧布局，使用安装前备份；再次升级新宿主后可继续读取原 v4。

发生恢复失败时保留私有备份并记录具体失败项，不能删除备份或宣布设备已恢复。候选包和测试脚本均不执行外部发布、推送或自动升级其他设备。
