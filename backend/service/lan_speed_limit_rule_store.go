package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
)

type LanSpeedLimitRuleStore interface {
	ReadRuleLists(ctx context.Context) ([]*models.LANCtrlSpeedLimitItem, []*models.LANCtrlSpeedLimitItem, error)
}

type defaultLanSpeedLimitRuleStore struct{}

var lanSpeedLimitRuleLoadConfig = uci.LoadConfig

func NewDefaultLanSpeedLimitRuleStore() LanSpeedLimitRuleStore {
	return &defaultLanSpeedLimitRuleStore{}
}

func preloadLanSpeedLimitRuleConfigs() {
	_ = lanSpeedLimitRuleLoadConfig("eqos", true)
	_ = lanSpeedLimitRuleLoadConfig("firewall", true)
}

func buildBlockedDeviceRule(mac, name, target string) (*models.LANCtrlSpeedLimitItem, bool) {
	normalizedMAC := strings.ToUpper(strings.TrimSpace(mac))
	if normalizedMAC == "" || !strings.HasPrefix(name, "BL_") || target != "REJECT" {
		return nil, false
	}
	return &models.LANCtrlSpeedLimitItem{
		Mac:           normalizedMAC,
		NetworkAccess: false,
		Enabled:       true,
	}, true
}

func buildSpeedLimitRule(ip, upload, download, comment string) *models.LANCtrlSpeedLimitItem {
	uploadSpeed, _ := strconv.ParseInt(upload, 10, 64)
	downloadSpeed, _ := strconv.ParseInt(download, 10, 64)
	return &models.LANCtrlSpeedLimitItem{
		IP:            ip,
		UploadSpeed:   uploadSpeed,
		DownloadSpeed: downloadSpeed,
		Comment:       comment,
		NetworkAccess: true,
	}
}

func (store *defaultLanSpeedLimitRuleStore) ReadRuleLists(ctx context.Context) ([]*models.LANCtrlSpeedLimitItem, []*models.LANCtrlSpeedLimitItem, error) {
	_ = ctx
	preloadLanSpeedLimitRuleConfigs()

	eqosSecs, ok := uci.GetSections("eqos", "device")
	if !ok {
		return nil, nil, errors.New("eqos device section not found")
	}
	firewallSecs, ok := uci.GetSections("firewall", "rule")
	if !ok {
		return nil, nil, errors.New("firewall rule section not found")
	}

	blocks := make([]*models.LANCtrlSpeedLimitItem, 0, len(firewallSecs))
	for _, sectionName := range firewallSecs {
		mac, _ := uci.GetLast("firewall", sectionName, "src_mac")
		name, _ := uci.GetLast("firewall", sectionName, "name")
		target, _ := uci.GetLast("firewall", sectionName, "target")
		if item, ok := buildBlockedDeviceRule(mac, name, target); ok {
			blocks = append(blocks, item)
		}
	}

	speedLimits := make([]*models.LANCtrlSpeedLimitItem, 0, len(eqosSecs))
	for _, sectionName := range eqosSecs {
		ip, _ := uci.GetLast("eqos", sectionName, "ip")
		upload, _ := uci.GetLast("eqos", sectionName, "upload")
		download, _ := uci.GetLast("eqos", sectionName, "download")
		comment, _ := uci.GetLast("eqos", sectionName, "comment")
		speedLimits = append(speedLimits, buildSpeedLimitRule(ip, upload, download, comment))
	}

	return blocks, speedLimits, nil
}
