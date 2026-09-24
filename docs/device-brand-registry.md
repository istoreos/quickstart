# 设备品牌注册表与识别规则治理

## 边界

品牌是经过审核的中性短文字，不是设备身份、产品类别或商标素材。注册表只把 OUI 等来源给出的原始厂商名称缩短为列表可读文字，不使用官方 logo、品牌字体或专属配色，也不因为厂商名称推断路由器。

分类的唯一生产入口是后端 `DeviceClassifier.Classify`。前端只消费分类契约；旧后端兼容适配器不得增加新的长期识别规则。

## 当前审核注册表

| 短品牌 | 接受的原始厂商范围 | 审核说明 |
| --- | --- | --- |
| ASUS | `ASUSTeK COMPUTER INC.`、`ASUS` | 仅公司全称和短名称；默认类别为电脑，路由器仍需型号/名称证据 |
| Apple | Apple 公司名及 `Inc.` 后缀变体 | 拒绝 Appleton 等近似名称 |
| Samsung | Samsung / Samsung Electronics 公司名变体 | 拒绝 Samsungton 等近似名称 |
| Xiaomi | Xiaomi 公司名、北京小米注册名称 | 仅显式公司名称 |
| HUAWEI | Huawei Technologies、华为注册名称 | 仅显式公司名称 |
| Synology | Synology 公司名及后缀变体 | 品牌不单独决定 NAS 类别 |
| QNAP | QNAP Systems 公司名 | 品牌不单独决定 NAS 类别 |
| HP | HP Inc.、Hewlett-Packard 公司名 | 两代公司拼写合并为用户熟悉的短名 |
| Canon | Canon 公司名及后缀变体 | 品牌不单独决定打印机类别 |
| Nintendo | Nintendo 公司名及后缀变体 | Switch 型号规则单独决定游戏设备 |
| Sony | Sony、Sony Interactive Entertainment | 品牌不单独决定产品类别 |
| Microsoft | Microsoft 公司名及后缀变体 | 拒绝虚拟网卡等组件描述 |
| Google | `Google LLC` | 只接受完整公司拼写 |
| Dell | Dell 公司名及后缀变体 | 品牌不单独决定电脑类别 |
| Lenovo | Lenovo 公司名及后缀变体 | 品牌不单独决定电脑类别 |
| LG | LG Electronics 公司名及后缀变体 | 品牌不单独决定电视类别 |

Intel、Realtek、Broadcom 等文本可能只代表网卡或芯片，当前不声明为整机品牌。未匹配厂商保留在详情的“原始厂商”字段，列表不擅自创建品牌。

## 变更规则

每次新增或放宽品牌/类别规则必须同时提交：

1. 至少一个真实格式的去隐私正例；
2. 至少一个近似名称、组件厂商或语义冲突反例；
3. 注册表审核说明；
4. fixture 预期的品牌、类别、来源和可信度；
5. 本地分类分布与 fallback 数量检查。

禁止使用无边界的宽泛子串。若无法确认整机品牌或类别，必须保留原始厂商并回退通用 PC。

## 受控样本与诊断

- 样本库：`backend/service/testdata/device_classification_fixtures.json`
- 契约测试：`TestDeviceClassifierReviewedFixtureLibrary`
- 当前基线：60 个样本、12 个类别/场景组（11 个可选类别 + fallback）、20 个以上原始厂商变体、10 个以上短品牌。
- 诊断输出仅包含样本数量、分组、来源分布、品牌数、厂商变体数和 fallback 数，不读取、保存或上传真实局域网设备数据。

运行：

```sh
cd backend
go test -v ./service -run TestDeviceClassifierReviewedFixtureLibrary
```

## 发布检查

- 所有 fallback 的输出类别必须为 `computer`，图标必须为原创通用 PC；
- `source` 和 `confidence` 不得为空；
- 品牌映射不得覆盖 `manufacturer` 原文；
- 人工分类只覆盖类别，不伪造品牌；
- 代码和发布资源不得加入第三方品牌 logo。
