# M24 设备资料、品牌与图标闭环验收

日期：2026-09-25

## 任务树结果

- [x] Device Profile v3 持久化 Alias、Manual Brand、Manual Category、Icon Mode 和 Manual Icon Key。
- [x] 品牌、类别和图标可以分别恢复自动，不互相误删。
- [x] 自动解析固定为手动图标、品牌+类型视觉、类型图标、通用电脑。
- [x] ASUS 网络设备与 ASUS 电脑具有不同形态，并以文字显示品牌，不包含商标 Logo。
- [x] 18 张新增原创 AI 辅助图标与既有 12 张组成 30 个稳定键。
- [x] manifest 记录键、中文无障碍名称、类别、文件和生成日期。

## 优化记录

1. Profile v3 后更新旧 v2 迁移断言并补齐品牌/图标持久化测试。
2. 联合检查发现一处 Markdown 尾随空格；修复后 `git diff --check` 通过。

## 证据

- 后端注册表与解析器：`backend/service/device_icon_registry.go`
- Profile 存储与 Module：`backend/service/device_classification_store.go`、`backend/service/device_profile.go`
- 资产与清单：`web/public/luci-static/quickstart/device-icons/`、`manifest.json`
- 治理：`docs/device-scene-icons.md`
- 测试结果：`docs/evidence/m24/test-results.txt`
