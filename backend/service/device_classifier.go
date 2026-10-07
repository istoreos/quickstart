package service

import (
	"regexp"
	"strings"

	"github.com/istoreos/quickstart/backend/models"
)

type DeviceClassificationInput struct {
	DisplayName  string
	Hostname     string
	Manufacturer string
	Model        string
}

type deviceClassifier interface {
	Classify(DeviceClassificationInput) *models.DeviceClassification
}

// DeviceClassifier is the single classification seam. Callers provide the
// evidence they have; rule priority, brand normalization and safe fallback
// remain inside the module.
type DeviceClassifier struct{}

type deviceCategoryRule struct {
	category string
	pattern  *regexp.Regexp
}

type reviewedDeviceBrand struct {
	brand      string
	pattern    *regexp.Regexp
	reviewNote string
}

// reviewedDeviceBrands is intentionally explicit. A match only shortens a
// manufacturer string for display; it never proves a product category.
var reviewedDeviceBrands = []reviewedDeviceBrand{
	{brand: "ASUS", pattern: regexp.MustCompile(`(?i)^(?:asustek computer inc\.?|asus)$`), reviewNote: "ASUSTeK corporate spelling and short form"},
	{brand: "Apple", pattern: regexp.MustCompile(`(?i)^apple(?:,? inc\.?)?$`), reviewNote: "Apple corporate suffix variants"},
	{brand: "Samsung", pattern: regexp.MustCompile(`(?i)^samsung(?: electronics(?: co\.,? ?ltd\.?)?)?$`), reviewNote: "Samsung Electronics corporate variants"},
	{brand: "Xiaomi", pattern: regexp.MustCompile(`(?i)^(?:xiaomi(?: communications? co\.,? ?ltd\.?)?|beijing xiaomi electronics co\.,? ?ltd\.?|北京小米.*)$`), reviewNote: "Xiaomi English and registered Chinese variants"},
	{brand: "HUAWEI", pattern: regexp.MustCompile(`(?i)^(?:huawei(?: technologies co\.,? ?ltd\.?)?|华为.*)$`), reviewNote: "Huawei English and registered Chinese variants"},
	{brand: "Synology", pattern: regexp.MustCompile(`(?i)^synology(?: incorporated| inc\.?)?$`), reviewNote: "Synology NAS manufacturer variants"},
	{brand: "QNAP", pattern: regexp.MustCompile(`(?i)^qnap systems,? inc\.??$`), reviewNote: "QNAP corporate spelling"},
	{brand: "HP", pattern: regexp.MustCompile(`(?i)^(?:hp inc\.?|hewlett[- ]packard(?: company)?)$`), reviewNote: "HP current and former corporate spellings"},
	{brand: "Canon", pattern: regexp.MustCompile(`(?i)^canon(?: inc\.?)?$`), reviewNote: "Canon corporate suffix variants"},
	{brand: "Nintendo", pattern: regexp.MustCompile(`(?i)^nintendo(?: co\.,? ?ltd\.?)?$`), reviewNote: "Nintendo corporate suffix variants"},
	{brand: "Sony", pattern: regexp.MustCompile(`(?i)^sony(?: interactive entertainment(?: inc\.?)?)?$`), reviewNote: "Sony consumer and interactive entertainment variants"},
	{brand: "Microsoft", pattern: regexp.MustCompile(`(?i)^microsoft(?: corporation)?$`), reviewNote: "Microsoft corporate suffix variants"},
	{brand: "Google", pattern: regexp.MustCompile(`(?i)^google llc$`), reviewNote: "Google corporate spelling"},
	{brand: "Dell", pattern: regexp.MustCompile(`(?i)^dell(?: inc\.?)?$`), reviewNote: "Dell corporate suffix variants"},
	{brand: "Lenovo", pattern: regexp.MustCompile(`(?i)^lenovo(?: group limited)?$`), reviewNote: "Lenovo corporate suffix variants"},
	{brand: "LG", pattern: regexp.MustCompile(`(?i)^lg electronics(?: inc\.?)?$`), reviewNote: "LG Electronics corporate suffix variants"},
}

var modelCategoryRules = []deviceCategoryRule{
	{category: "network", pattern: regexp.MustCompile(`(?i)\b(?:rt|gt)-?ax\d+[a-z0-9-]*\b|\bzenwifi\b`)},
	{category: "gaming", pattern: regexp.MustCompile(`(?i)\bnintendo[-_ ]?switch\b|\bplaystation\s?[345]?\b|\bxbox\b|\bsteam[-_ ]?deck\b`)},
	{category: "wearable", pattern: regexp.MustCompile(`(?i)\bapple[-_ ]?watch\b|\bgalaxy[-_ ]?watch\b`)},
}

var hostnameCategoryRules = []deviceCategoryRule{
	{category: "storage", pattern: regexp.MustCompile(`(?i)\bnas\b|\bserver\b|storage|truenas|openmediavault|synology|qnap|群晖|威联通`)},
	{category: "camera", pattern: regexp.MustCompile(`(?i)camera|webcam|doorbell|\bipc[-_ ]|cctv|摄像头|监控|门铃`)},
	{category: "gaming", pattern: regexp.MustCompile(`(?i)gaming|game[-_ ]?console|游戏|游戏主机`)},
	{category: "printer", pattern: regexp.MustCompile(`(?i)printer|laserjet|deskjet|officejet|打印机|一体机`)},
	{category: "wearable", pattern: regexp.MustCompile(`(?i)smart[-_ ]?watch|fitness[-_ ]?(?:band|tracker)|wearable|手表|手环`)},
	{category: "tv", pattern: regexp.MustCompile(`(?i)(?:^|[-_ ])tv(?:$|[-_ ])|television|smart[-_ ]?tv|chromecast|streaming[-_ ]?(?:box|stick)|电视|投影`)},
	{category: "tablet", pattern: regexp.MustCompile(`(?i)tablet|\bipad\b|e[-_ ]?reader|kindle|平板|阅读器`)},
	{category: "phone", pattern: regexp.MustCompile(`(?i)phone|iphone|android|mobile|pixel[-_ ]?\d|手机`)},
	{category: "network", pattern: regexp.MustCompile(`(?i)router|gateway|openwrt|access[-_ ]?point|\bmesh\b|repeater|\bswitch\b|路由|网关|交换机|无线接入点`)},
	{category: "smart-home", pattern: regexp.MustCompile(`(?i)smart[-_ ]?home|home[-_ ]?assistant|speaker|thermostat|sensor|vacuum|air[-_ ]?purifier|light[-_ ]?(?:bulb|strip)|smart[-_ ]?plug|智能家居|音箱|传感器|扫地|灯泡|插座`)},
	{category: "computer", pattern: regexp.MustCompile(`(?i)desktop|laptop|notebook|workstation|macbook|imac|windows|ubuntu|\bpc\b|电脑|笔记本|工作站`)},
}

func NewDeviceClassifier() *DeviceClassifier { return &DeviceClassifier{} }

func (classifier *DeviceClassifier) Classify(input DeviceClassificationInput) *models.DeviceClassification {
	manufacturer := strings.TrimSpace(input.Manufacturer)
	brand := normalizedDeviceBrand(manufacturer)
	modelEvidence := strings.TrimSpace(strings.Join([]string{input.Model, input.DisplayName, input.Hostname}, " "))
	for _, rule := range modelCategoryRules {
		if rule.pattern.MatchString(modelEvidence) {
			return deviceClassification(brand, manufacturer, rule.category, "model", "high")
		}
	}

	nameEvidence := strings.TrimSpace(strings.Join([]string{input.DisplayName, input.Hostname}, " "))
	for _, rule := range hostnameCategoryRules {
		if rule.pattern.MatchString(nameEvidence) {
			return deviceClassification(brand, manufacturer, rule.category, "hostname", "medium")
		}
	}

	if brand == "ASUS" {
		return deviceClassification(brand, manufacturer, "network", "manufacturer_default", "medium")
	}
	return deviceClassification(brand, manufacturer, "computer", "fallback", "low")
}

func normalizedDeviceBrand(manufacturer string) string {
	value := strings.TrimSpace(manufacturer)
	for _, reviewed := range reviewedDeviceBrands {
		if reviewed.pattern.MatchString(value) {
			return reviewed.brand
		}
	}
	return ""
}

func deviceClassification(brand, manufacturer, category, source, confidence string) *models.DeviceClassification {
	return &models.DeviceClassification{
		Brand: brand, Manufacturer: manufacturer, Category: category, Source: source, Confidence: confidence,
	}
}

func validDeviceCategory(category string) bool {
	switch category {
	case "phone", "computer", "tablet", "tv", "network", "smart-home", "camera", "gaming", "storage", "printer", "wearable":
		return true
	default:
		return false
	}
}
