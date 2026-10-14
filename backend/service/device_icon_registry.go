package service

import (
	"strings"

	"github.com/istoreos/quickstart/backend/models"
)

type deviceIconDescriptor struct {
	Key      string
	Label    string
	Category string
}

var deviceIconRegistry = []deviceIconDescriptor{
	{Key: "computer", Label: "通用电脑", Category: "computer"},
	{Key: "phone", Label: "手机", Category: "phone"},
	{Key: "tablet", Label: "平板电脑", Category: "tablet"},
	{Key: "tv", Label: "电视", Category: "tv"},
	{Key: "network", Label: "无线路由器", Category: "network"},
	{Key: "smart-home", Label: "智能家居", Category: "smart-home"},
	{Key: "camera", Label: "摄像头", Category: "camera"},
	{Key: "gaming", Label: "游戏设备", Category: "gaming"},
	{Key: "storage", Label: "网络存储", Category: "storage"},
	{Key: "printer", Label: "打印机", Category: "printer"},
	{Key: "wearable", Label: "穿戴设备", Category: "wearable"},
	{Key: "unknown", Label: "未知设备", Category: "computer"},
	{Key: "desktop", Label: "台式电脑", Category: "computer"},
	{Key: "laptop", Label: "笔记本电脑", Category: "computer"},
	{Key: "smart-speaker", Label: "智能音箱", Category: "smart-home"},
	{Key: "smart-bulb", Label: "智能灯", Category: "smart-home"},
	{Key: "thermostat", Label: "温控器", Category: "smart-home"},
	{Key: "sensor", Label: "传感器", Category: "smart-home"},
	{Key: "door-lock", Label: "智能门锁", Category: "smart-home"},
	{Key: "robot-vacuum", Label: "扫地机器人", Category: "smart-home"},
	{Key: "air-conditioner", Label: "空调", Category: "smart-home"},
	{Key: "projector", Label: "投影仪", Category: "tv"},
	{Key: "set-top-box", Label: "电视盒子", Category: "tv"},
	{Key: "game-console", Label: "游戏主机", Category: "gaming"},
	{Key: "handheld-game", Label: "掌上游戏机", Category: "gaming"},
	{Key: "home-server", Label: "家庭服务器", Category: "storage"},
	{Key: "network-switch", Label: "网络交换机", Category: "network"},
	{Key: "access-point", Label: "无线接入点", Category: "network"},
	{Key: "network-bridge", Label: "网络桥接器", Category: "network"},
	{Key: "e-reader", Label: "电子阅读器", Category: "tablet"},
}

func validDeviceIconKey(key string) bool {
	for _, item := range deviceIconRegistry {
		if item.Key == key {
			return true
		}
	}
	return false
}

func deviceIconForProfile(classification *models.DeviceClassification, record deviceProfileRecord) *models.DeviceIconVisual {
	category := "computer"
	brand := ""
	if classification != nil {
		if validDeviceCategory(classification.Category) {
			category = classification.Category
		}
		brand = strings.TrimSpace(classification.Brand)
	}
	if record.IconMode == "manual" && validDeviceIconKey(record.IconKey) {
		descriptor := lookupDeviceIcon(record.IconKey)
		return &models.DeviceIconVisual{Mode: "manual", PreferenceKey: record.IconKey, ResolvedKey: "manual:" + record.IconKey, AssetKey: record.IconKey, Label: descriptor.Label}
	}
	assetKey := category
	if !validDeviceIconKey(assetKey) {
		assetKey = "computer"
	}
	resolvedKey := "category:" + assetKey
	brandLabel := ""
	if brand != "" {
		resolvedKey = "brand:" + strings.ToLower(strings.ReplaceAll(brand, " ", "-")) + ":" + assetKey
		brandLabel = brand
	}
	label := lookupDeviceIcon(assetKey).Label
	if brandLabel != "" {
		label = brandLabel + " · " + label
	}
	return &models.DeviceIconVisual{Mode: "auto", ResolvedKey: resolvedKey, AssetKey: assetKey, Label: label, BrandLabel: brandLabel}
}

func lookupDeviceIcon(key string) deviceIconDescriptor {
	for _, item := range deviceIconRegistry {
		if item.Key == key {
			return item
		}
	}
	return deviceIconDescriptor{Key: "computer", Label: "通用电脑", Category: "computer"}
}
