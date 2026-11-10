# FastNet 0.8.8 提交与发布

用户授权：提交卡片改动，增加版本号，发布新版本并更新 istoreos-app-hub/apps/fastnet。

## 完成

- FastNet 提交并推送 `d5063d718233d723769271fb33455e046aff32b8`，标签 `v0.8.8`。
- 插件目录提交并推送 `8d1716e353c511c6586e5935bfa4251a6dec1f9c`；运行 0.8.8-1、LuCI 0.8.8-r1、元包 0.8.8-17、卡片 0.8.8-1。本轮已有 meta release 17，没有再增加第二次。
- GitHub 正式发布：https://github.com/istoreos/istoreos-app-hub/releases/tag/fastnet-runtime-v0.8.8 。该发布标签对应本次插件提交；7 个附件全部 uploaded，GitHub digest 与本地 SHA256 逐项一致，公开下载的运行归档再次核验通过。
- 正式归档 SHA256：`c2ce53468e19876c92abdca33358476aeaa15404708e8f45d413c7664bf3fa94`，已写入 apps/fastnet/fastnet/Makefile；插件资源 provenance 指向本次 FastNet 源码提交及逐文件哈希。
- 遵循 FastNet Makefile，经 SSD 流程构建 x86_64、aarch64、ARMv7。压缩规则保持原样，提供原始归档。Go 全包、LAN 行为/边界/顺序、包契约、UPX 完整性及正式二进制启动/version/health/空历史检查通过；重新构建的前端与内嵌资源逐文件一致。18 项卡片浏览器检查沿用本轮相同卡片字节的结果。
- 项目原发布脚本 dry-run 通过后将官方下载源站 root@192.168.3.1 的 FastNet 目录切换到 0.8.8；原 0.8.1 元数据与安装脚本有独立回退备份，见 publication.json。

## 公开镜像限制

`dl.istoreos.com` 当前跳转至 `fw.koolcenter.com`，后者及另外两处下载入口仍返回 0.8.1 元数据，新文件在检查时返回 404；添加 no-cache 及查询参数仍相同，因此不能声称公网镜像已更新。旧文件 last-modified 为 2026-11-09。镜像维护 SSH 入口未通过主机密钥校验，未绕过校验或修改其配置。

当前插件配方从 GitHub Release 下载已公开验证的归档，不依赖上述镜像。源站发布与公网镜像同步分别记录。本次没有同步 legacy 仓库、触发基于旧 feeds 的 Build IPKs 或上架 iStore 软件源；用户指定的聚合插件目录已经更新且可通过新归档构建。

本轮没有更新测速来源目录服务，也没有再替换 192.168.30.7 的核心；该机保留前轮验收的 0.8.7 核心及新卡片布局。正式产物与唯一验收记录已放回持久仓库目录，不只保存在 SSD 缓存。
