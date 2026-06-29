package service

import (
	"context"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
)

type LanSpeedLimitedDeviceInventorySnapshot struct {
	DeviceByMAC map[string]*models.DeviceInfo
	DeviceByIP  map[string]*models.DeviceInfo
}

type speedLimitHostRecord struct {
	MAC  string
	Name string
}

type LanSpeedLimitedDeviceInventoryReader interface {
	ReadInventory(ctx context.Context) (LanSpeedLimitedDeviceInventorySnapshot, error)
}

type SpeedLimitHostnameReader interface {
	ReadHostnames(ctx context.Context) (map[string]string, error)
}

type defaultLanSpeedLimitedDeviceInventoryReader struct{}

type defaultSpeedLimitHostnameReader struct{}

var lanSpeedLimitedDeviceListLoadConfig = uci.LoadConfig
var lanSpeedLimitedDeviceListGetSections = uci.GetSections
var lanSpeedLimitedDeviceListGetLast = uci.GetLast

func NewDefaultLanSpeedLimitedDeviceInventoryReader() LanSpeedLimitedDeviceInventoryReader {
	return &defaultLanSpeedLimitedDeviceInventoryReader{}
}

func NewDefaultSpeedLimitHostnameReader() SpeedLimitHostnameReader {
	return &defaultSpeedLimitHostnameReader{}
}

func buildLanSpeedLimitedDeviceInventory(devices []*models.DeviceInfo) LanSpeedLimitedDeviceInventorySnapshot {
	snapshot := LanSpeedLimitedDeviceInventorySnapshot{
		DeviceByMAC: map[string]*models.DeviceInfo{},
		DeviceByIP:  map[string]*models.DeviceInfo{},
	}
	for _, item := range devices {
		if item == nil || item.Mac == "" || item.Ipv4addr == "" {
			continue
		}
		mac := strings.ToUpper(strings.TrimSpace(item.Mac))
		snapshot.DeviceByMAC[mac] = item
		snapshot.DeviceByIP[item.Ipv4addr] = item
	}
	return snapshot
}

func buildSpeedLimitHostnameMap(records []speedLimitHostRecord) map[string]string {
	hostnames := make(map[string]string, len(records))
	for _, record := range records {
		if record.MAC == "" {
			continue
		}
		hostnames[strings.ToUpper(strings.TrimSpace(record.MAC))] = record.Name
	}
	return hostnames
}

func (reader *defaultLanSpeedLimitedDeviceInventoryReader) ReadInventory(ctx context.Context) (LanSpeedLimitedDeviceInventorySnapshot, error) {
	resp, err := NetworkDeviceList(ctx)
	if err != nil {
		return LanSpeedLimitedDeviceInventorySnapshot{}, err
	}
	if resp == nil || resp.Result == nil {
		return buildLanSpeedLimitedDeviceInventory(nil), nil
	}
	return buildLanSpeedLimitedDeviceInventory(resp.Result.Devices), nil
}

func (reader *defaultSpeedLimitHostnameReader) ReadHostnames(ctx context.Context) (map[string]string, error) {
	_ = ctx
	lanSpeedLimitedDeviceListLoadConfig("dhcp", true)

	hostSections, ok := lanSpeedLimitedDeviceListGetSections("dhcp", "host")
	if !ok {
		return map[string]string{}, nil
	}

	records := make([]speedLimitHostRecord, 0, len(hostSections))
	for _, sectionName := range hostSections {
		mac, _ := lanSpeedLimitedDeviceListGetLast("dhcp", sectionName, "mac")
		name, _ := lanSpeedLimitedDeviceListGetLast("dhcp", sectionName, "name")
		records = append(records, speedLimitHostRecord{
			MAC:  mac,
			Name: name,
		})
	}

	return buildSpeedLimitHostnameMap(records), nil
}
