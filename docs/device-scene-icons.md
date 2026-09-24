# 设备场景图标体系

## 目标

设备中心使用原创、非品牌化的设备场景图标，帮助用户快速分辨设备用途。图标不替代厂商识别，也不展示或仿制厂商商标。

## 当前场景

| 场景键 | 用户语义 | 识别线索示例 |
| --- | --- | --- |
| `phone` | 手机 | phone、iPhone、Android、手机 |
| `computer` | 电脑 | desktop、laptop、workstation、电脑 |
| `tablet` | 平板/阅读器 | tablet、iPad、e-reader、平板 |
| `tv` | 电视与影音 | TV、television、streaming box、电视、投影 |
| `network` | 网络设备 | router、gateway、OpenWrt、mesh、交换机 |
| `smart-home` | 智能家居 | speaker、sensor、vacuum、灯泡、插座 |
| `camera` | 摄像头 | camera、doorbell、IPC、监控 |
| `gaming` | 游戏设备 | game console、PlayStation、Xbox、Nintendo |
| `storage` | 存储与服务器 | NAS、server、storage、群晖、威联通 |
| `printer` | 打印机 | printer、LaserJet、打印机、一体机 |
| `wearable` | 穿戴设备 | smart watch、fitness band、手表、手环 |
| `unknown` | 未知设备资产位 | 兼容旧资源；实际无足够线索时显示通用电脑图标 |

识别顺序会优先处理含义更具体的场景。例如游戏设备先于网络设备判断，避免设备名中的 `switch` 被误判为交换机。当前规则只使用设备名称、主机名、型号线索和厂商文本，不会把推测写回设备配置。无可靠证据时分类结果为 `computer/fallback`，不得默认显示路由器。

## 生成与合规约束

- 生成方式：OpenAI 内置图像生成工具，2026-09-24。
- 统一提示词：原创、适合 32–48px 的圆角几何设备图标；薰衣草紫、靛蓝和少量薄荷绿；透明背景；无文字、字母、商标、水印或品牌特征；不得仿制商业产品轮廓。
- 每个场景使用独立提示词生成，并由人工检查可辨识度、透明通道和品牌元素。
- 原始输出转换为最大边 256px 的透明 WebP。12 张交付图总计约 68KB。
- 图标表达的是通用设备类别，不代表真实厂商或精确设备型号。无法可靠判断时必须使用 `unknown`。

生成式图像只能降低与现有品牌视觉冲突的风险，不能构成绝对的不侵权保证。正式对外发布前仍应保留来源记录，并由产品或法务完成最终审查。

## 工程位置

- 交付图标：`web/public/luci-static/quickstart/device-icons/`
- 场景类型和图标路径：`web/src/pages/device/deviceScene.ts`
- 识别规则：`backend/service/device_classifier.go`
- 品牌注册表与治理记录：`docs/device-brand-registry.md`
- 展示组件：`web/src/pages/device/components/deviceSceneIcon.vue`

列表、移动卡片和详情抽屉共用同一场景类型与展示组件，避免三处视觉或语义漂移。
