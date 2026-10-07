# 局域网设备管理低保真原型

> 状态：review  
> 创建日期：2026-09-25  
> 入口：[index.html](./index.html)

## 当前覆盖

原型已逐项对照产品分析草稿，覆盖设备发现与身份、单设备管理、分组与计划、局域网基础能力、异常与专家诊断五个区域，共整理 48 项产品能力：47 项进入用户界面，1 项明确保持后端内部。详细状态见[产品能力覆盖矩阵](./coverage.md)。

DHCP tag、option 3/6、UCI 等实现细节按产品原则不进入普通界面；真实配置写入、流量采样和故障切换也不会在低保真中执行。

## 验证目标

验证局域网设备管理是否能够以“设备”为核心，同时容纳：

- 地址保留；
- 中文设备备注名；
- 独立且安全的 DHCP 网络主机名；
- 厂商、设备类型与用户自选图标的优先关系；
- 单设备限速、暂停联网和计划；
- 旁路由、浮动网关和指定 IP；
- 浮动网关或限速能力缺失时的局部降级与恢复；
- 主路由、旁路由与 DHCP 分配权差异；
- 桌面和移动端的一致任务路径。

上游产品文档：

- [产品分析草稿](../../lan-device-management-product-analysis-draft.md)
- [低保真交互设计](../../lan-device-management-low-fidelity-design.md)
- [产品能力覆盖矩阵](./coverage.md)

## 原型方案

- `variant=A`：设备优先 + 详情抽屉，当前推荐；
- `variant=B`：列表 + 常驻详情双栏；
- `variant=C`：任务引导首页。

状态参数：

- `screen=device|groups|network`：切换产品区域；
- `context=main|side-local|side-external|ambiguous`：模拟拓扑位置与 DHCP 分配权组合；
- `caps=all|no-floating|no-speed|none|error`：模拟可选能力可用、未安装或异常；
- `device=asus|tv|unknown|planned`：直接打开一个示例设备详情；
- `modal=profile|network|limits|usage|group|dhcp|routes|floating|diagnostics|capability-setup`：直接打开主要编辑或诊断流程；
- `capability=floating|speed`：在能力说明层指定浮动网关或设备限速；
- 页面底部和顶部均提供对应切换控件。

示例：

```text
?variant=A&screen=device&context=main&device=asus
?variant=A&screen=network&context=side-external
?variant=A&screen=device&caps=no-floating&device=asus&modal=network
?variant=A&screen=device&caps=no-speed&device=asus&modal=limits
```

## 运行方式

在项目根目录执行：

```bash
./scripts/run-lowfi-server.sh start
```

访问：

```text
http://<开发机局域网IP>:5173/lan-device-management/
```

管理服务：

```bash
./scripts/run-lowfi-server.sh status
./scripts/run-lowfi-server.sh restart
./scripts/run-lowfi-server.sh stop
```

## 当前已确认语义

- DNS 默认跟随所选上网路线；
- 本机无 DHCP 分配权时，设备路线只读并引导到主路由；
- 路线能否编辑由 DHCP 分配权决定，拓扑角色用于解释和默认推荐；
- 中文设备备注名不写入 DHCP；
- 网络主机名位于高级设置且只允许安全 ASCII；
- 图标优先级为“用户手动选择 > 厂商原创视觉 > 设备类型 > 通用电脑”，并允许恢复自动推荐；
- 可选能力缺失时只禁用依赖它的选项，保留已有期望配置，其他设置继续可用。

## 待评审

- 是否采用方案 A 作为正式布局；
- 桌面抽屉、移动全屏的详情形态；
- 设备列表最多展示两个策略或异常摘要；
- 局域网设置的五个分区是否完整。

评审完成后，把结论更新到低保真交互设计文档，并将本文件状态改为 `confirmed` 或 `archived`。

## 迭代记录

- 2026-09-25：建立 A/B/C 三种整体结构；
- 2026-09-25：补齐设备资料、添加设备、地址与路线、限速/时段/额度、分组、路由器上下文、DHCP、路线目标、浮动网关、统计和诊断流程；
- 2026-09-25：增加主路由/旁路由与本机/外部 DHCP 组合、等待续租、应用失败和恢复演示状态；
- 2026-09-25：增加 30 个自选图标、自动图标来源说明，以及浮动网关/设备限速缺失、异常、安装恢复场景。
