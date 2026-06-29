package service

import (
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
)

type LanDeviceSnapshot struct {
	IP  string
	Mac string
}

type HostHintSnapshot struct {
	Hostname string
}

type TrafficStatSnapshot struct {
	UploadSpeed   int64
	DownloadSpeed int64
}

type StaticAssignmentSnapshot struct {
	Device *models.LANStaticAssigned
}

type SpeedLimitSnapshot struct {
	Device *models.LANCtrlSpeedLimitItem
}

type LanDeviceAggregateState struct {
	Devices           models.LANDevices
	DhcpTags          []*models.LANCtrlDhcpTagInfo
	HostHints         map[string]HostHintSnapshot
	WifiMACs          map[string]struct{}
	TrafficStats      map[string]TrafficStatSnapshot
	StaticAssignments map[string]*models.LANStaticAssigned
	BlockMap          map[string]*models.LANCtrlSpeedLimitItem
	SpeedLimitMap     map[string]*models.LANCtrlSpeedLimitItem
}

func buildDeviceInventoryItem(ip, mac string) (*models.LANDevice, bool) {
	ip = strings.TrimSpace(ip)
	mac = strings.ToUpper(strings.TrimSpace(mac))
	if ip == "" || mac == "" {
		return nil, false
	}

	item := &models.LANDevice{
		IP:             ip,
		Mac:            mac,
		Vendor:         GomanufSearch(strings.TrimSpace(mac)),
		Intr:           "lan",
		StaticAssigned: &models.LANStaticAssigned{},
		SpeedLimit:     &models.LANCtrlSpeedLimitItem{},
	}
	item.StaticAssigned.AssignedMac = item.Mac
	item.StaticAssigned.AssignedIP = item.IP
	item.SpeedLimit.IP = item.IP
	item.SpeedLimit.Mac = item.Mac
	item.SpeedLimit.NetworkAccess = true

	return item, true
}
