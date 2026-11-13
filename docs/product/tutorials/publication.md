# 首期发布文案与配套操作说明

## 标题

FastNet 更新了：测测你家宽带，再把成绩放到 iStoreOS 首页

## 视频简介

新版 FastNet 使用多来源并发下载，减少单个下载节点对结果的影响。
这次实际测到了约 921.4 Mbps 下载、65.6 Mbps 上传；这些数值只代表本次
网络环境。跟着视频完成安装或更新、一次测速，再把最近结果放到首页。

软件：FastNet 0.8.9；Quickstart BETA 1113-0.17.1-rc20261028-r2。
安装演示使用 ARM64 测试机，测速和卡片演示使用 x86 测试机。
浏览器在 Windows 视频服务器运行；内网结果对应这台浏览器到测试路由器的路径。
本次设备 NAT 检测超时、IPv6 未就绪，画面如实保留；不代表所有检测项成功。
安装与测速等待经过剪辑，未生成或替换软件界面、测试数字。

## 安装或更新 FastNet

在 iStore 软件中心找到 FastNet，选择安装或更新；也可以在路由器终端以 root 执行：

```sh
is-opkg update
is-opkg install app-meta-fastnet
```

完成后打开 FastNet，核对版本和服务状态，点击“开始全链路体检”。
外网下载/上传与浏览器到路由器的内网测速是两条不同链路，不要混为同一带宽。

## 更新 Quickstart BETA

[官方 BETA 目录](https://fw.koolcenter.com/iStoreOS/alpha/quickstart/1113-0.17.1-rc20261028-r2/)
提供版本清单、安装器，以及 x86_64/ARM64 对应安装包。这是测试版入口。

先下载并查看脚本，再运行只读检查：

```sh
curl -fL https://fw.koolcenter.com/iStoreOS/alpha/quickstart/1113-0.17.1-rc20261028-r2/install_quickstart.sh -o /tmp/install_quickstart.sh
sh /tmp/install_quickstart.sh --check
```

确认适配后，先在外部存储上准备真实、完整的系统备份，再按安装器说明更新。
下面的路径只是示意，必须换成自己已经存在的备份文件：

```sh
sh /tmp/install_quickstart.sh --install --backup-file /mnt/EXTERNAL/system-backup.tar.gz
```

官方安装器要求两次交互确认。本次拍摄设备已经安装该 BETA，只执行了
`--check`，没有为了镜头重复安装同版本。

## 添加首页测速卡片

1. 刷新 Quickstart 首页，点右上角“自定义首页”。
2. 点“添加模块”，选择“外网测速”。
3. 点“确认放置”，再点“完成”。
4. 如果出现“完成前整理”，确认整理并完成。
5. 刷新页面，检查卡片仍在。

卡片展示最近有效结果；WAN、LAN、NAT 和 IPv6 的测量时间可能不同。
打开测速界面不会自动发起新测试，需要在 FastNet 中点击开始。

## 互动文案

你家宽带能跑多少？测一次，把成绩留在首页，也欢迎晒出你的测速结果。

## 素材状态

已生成本地审阅成片，尚未上传视频平台或对外发布。
