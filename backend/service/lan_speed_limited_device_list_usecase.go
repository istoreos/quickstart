package service

import (
	"context"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
)

type LanSpeedLimitedDeviceListService struct {
	RuleStore       LanSpeedLimitRuleStore
	InventoryReader LanSpeedLimitedDeviceInventoryReader
	HostnameReader  SpeedLimitHostnameReader
}

func NewLanSpeedLimitedDeviceListService() *LanSpeedLimitedDeviceListService {
	return &LanSpeedLimitedDeviceListService{
		RuleStore:       NewDefaultLanSpeedLimitRuleStore(),
		InventoryReader: NewDefaultLanSpeedLimitedDeviceInventoryReader(),
		HostnameReader:  NewDefaultSpeedLimitHostnameReader(),
	}
}

func normalizeLanSpeedLimitedDeviceMAC(mac string) string {
	return strings.ToUpper(strings.TrimSpace(mac))
}

func (svc *LanSpeedLimitedDeviceListService) GetListSpeedLimitedDevices(ctx context.Context) (*models.LANCtrlSpeedLimitResponse, error) {
	blocks, speedLimits, err := svc.RuleStore.ReadRuleLists(ctx)
	if err != nil {
		return nil, err
	}

	inventory := LanSpeedLimitedDeviceInventorySnapshot{}
	if svc.InventoryReader != nil {
		if snapshot, err := svc.InventoryReader.ReadInventory(ctx); err == nil {
			inventory = snapshot
		}
	}

	result := mergeLanSpeedLimitedDeviceLists(blocks, speedLimits, inventory)

	hostnames := map[string]string{}
	if svc.HostnameReader != nil {
		if values, err := svc.HostnameReader.ReadHostnames(ctx); err == nil {
			hostnames = values
		}
	}
	attachLanSpeedLimitedHostnames(result, hostnames)

	return &models.LANCtrlSpeedLimitResponse{Result: result}, nil
}

func mergeLanSpeedLimitedDeviceLists(blocks, speedLimits []*models.LANCtrlSpeedLimitItem, inventory LanSpeedLimitedDeviceInventorySnapshot) []*models.LANCtrlSpeedLimitItem {
	result := make([]*models.LANCtrlSpeedLimitItem, 0, len(speedLimits)+len(blocks))
	remainingBlocks := make(map[string]*models.LANCtrlSpeedLimitItem, len(blocks))
	for _, block := range blocks {
		if block == nil {
			continue
		}
		mac := normalizeLanSpeedLimitedDeviceMAC(block.Mac)
		if mac == "" {
			continue
		}
		copyItem := *block
		copyItem.Mac = mac
		remainingBlocks[mac] = &copyItem
	}

	for _, speedItem := range speedLimits {
		if speedItem == nil {
			continue
		}
		copyItem := *speedItem
		if device := inventory.DeviceByIP[speedItem.IP]; device != nil {
			copyItem.Mac = normalizeLanSpeedLimitedDeviceMAC(device.Mac)
			if _, ok := remainingBlocks[copyItem.Mac]; ok {
				copyItem.NetworkAccess = false
				delete(remainingBlocks, copyItem.Mac)
			}
		}
		result = append(result, &copyItem)
	}

	for _, block := range blocks {
		if block == nil {
			continue
		}
		mac := normalizeLanSpeedLimitedDeviceMAC(block.Mac)
		remaining := remainingBlocks[mac]
		if remaining == nil {
			continue
		}
		device := inventory.DeviceByMAC[mac]
		if device == nil {
			continue
		}
		copyItem := *remaining
		copyItem.IP = device.Ipv4addr
		result = append(result, &copyItem)
	}

	return result
}

func attachLanSpeedLimitedHostnames(items []*models.LANCtrlSpeedLimitItem, hostnames map[string]string) {
	for _, item := range items {
		if item == nil {
			continue
		}
		item.Hostname = hostnames[normalizeLanSpeedLimitedDeviceMAC(item.Mac)]
	}
}
