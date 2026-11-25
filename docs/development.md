# 开发指南

Quickstart 由 Go 后端和 Vue 3 前端组成，正式运行依赖 iStoreOS/OpenWrt 的 LuCI 与系统服务。
本地构建成功不代表设备上的网络、存储或可选组件已完成验证。

## 环境与依赖

- Go 版本以 [`backend/go.mod`](../backend/go.mod) 为准：当前声明 Go 1.23.0，工具链为 1.23.5。
- 前端需要 Node.js 与 npm，使用 [`web/package-lock.json`](../web/package-lock.json) 安装锁定依赖。
- 打包前端需要 `zip`；后端构建脚本使用 Linux shell 工具。

在仓库根目录执行：

```sh
npm --prefix web ci
```

Go 命令会按 `go.mod` / `go.sum` 获取所需依赖。Quickstart 的构建、测试和发布不依赖
`third_party/` 下的源码；该目录仅用于本地参考或独立构建，已加入 Git 忽略列表。

Bandix 是设备端可选服务，Quickstart 通过 HTTP API 与其交互。如需研究或单独构建，
可自行克隆上游 [Bandix](https://github.com/timsaya/bandix)、
[OpenWrt 包](https://github.com/timsaya/openwrt-bandix)和
[LuCI 应用](https://github.com/timsaya/luci-app-bandix)到 `third_party/bandix/`，
并保留各自的许可证和 NOTICE。历史版本与架构说明见
[Bandix 源码走读](archive/bandix-source-walkthrough.md)。

## 本地检查与构建

在仓库根目录执行：

```sh
make test
node --test web/test/*.test.mjs
npm --prefix web run tsc
make build-amd64
make build-web
```

`make test` 执行后端 Go 测试。前端回归用例当前位于 `web/test/*.test.mjs`，直接使用上面的
Node 命令运行。部分用例会加载 TypeScript，因此需先安装前端依赖。

`make build-amd64` 默认输出 `bin/quickstart.amd64`；`make build` 同时构建 Linux amd64、
arm64 和 armv7。可用 `BUILD_DIR` 指定后端产物目录。

`make build-web` 生成 `web/dist/luci-static/` 和 `web/dist/quickstart_web.zip`，不包含类型检查；
`npm --prefix web run build` 会先检查类型再构建。

更多构建与运维命令见 `make help`；远程检查需要开发者自己的测试设备配置。

## 前端联调

```sh
npm --prefix web run dev
```

Vite 配置端口为 `8301`，将 `/cgi-bin/luci/istore/` 代理至 `http://127.0.0.1:8300`。
该代理目标需另行接入测试设备的 LuCI/API 链路；它不是后端默认监听地址。
后端 `quickstart serve` 默认通过 `/var/run/quickstart/http.sock` 提供 HTTP API，
不再监听 TCP 3038；可通过 `--http-unix` 指定其他 socket 路径。LuCI 的
`istore_backend.lua` 连接该 socket，浏览器的 API 地址及 SID/token 鉴权保持不变。
设备原有的 `--unix /var/run/quickstart/local.sock` 继续提供 DHCP/DHNS、LCD 内部接口。
后端与 LuCI 控制器需要配套更新；原来直连 3038 的运维脚本应改走 LuCI，
或使用 `curl --unix-socket /var/run/quickstart/http.sock http://localhost/...` 并提供有效鉴权头。
标准部署通过 Unix socket 检查 HTTP 服务是否可达，不使用伪造的登录态；目标设备需要支持
Unix socket 的 curl。部署前会检查设备已有的后端和 Lua 控制器均使用 socket，确保失败时
可以安全回滚。首次从 TCP 迁移需要成对更新后端与 Lua；标准回滚也会拒绝恢复 TCP 版后端。

正式前端还依赖 LuCI 宿主提供的登录态、`window.vue_base`、`window.vue_lang` 和
`window.vue_lang_data`。单独打开 Vite 页面不能替代完整设备联调；LuCI 挂载模板位于
[独立集成仓库](https://github.com/linkease/nas-packages-luci/tree/main/luci/luci-app-quickstart)。

独立设计预览使用 `web/vite.design.config.ts` 和 `web/src/design/`，通过
`./scripts/ui-design.sh start|status|stop` 管理。预览使用模拟数据，不转发路由器请求，
不能据此判断实机功能是否可用。
