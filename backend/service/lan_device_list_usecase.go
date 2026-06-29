package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type LanDeviceListService struct {
	LanStatusReader        LanStatusReader
	InventoryReader        DeviceInventoryReader
	DhcpTagReader          DhcpTagReader
	HostHintReader         HostHintReader
	WifiAssocReader        WifiAssocReader
	TrafficStatReader      TrafficStatReader
	StaticAssignmentReader StaticAssignmentReader
	SpeedLimitReader       LanDeviceSpeedLimitReader
}

func NewLanDeviceListService() *LanDeviceListService {
	return &LanDeviceListService{
		LanStatusReader:        NewDefaultLanStatusReader(),
		InventoryReader:        NewDefaultDeviceInventoryReader(),
		DhcpTagReader:          NewDefaultDhcpTagReader(),
		HostHintReader:         NewDefaultHostHintReader(),
		WifiAssocReader:        NewDefaultWifiAssocReader(),
		TrafficStatReader:      NewDefaultTrafficStatReader(),
		StaticAssignmentReader: NewDefaultStaticAssignmentReader(),
		SpeedLimitReader:       NewDefaultLanDeviceSpeedLimitReader(),
	}
}

func (svc *LanDeviceListService) GetListDevices(ctx context.Context, serviceBackend *ServiceBackend) (*models.LANDeviceResponse, error) {
	lanStatus, err := svc.LanStatusReader.ReadLanStatus(ctx)
	if err != nil {
		return nil, err
	}

	devices, err := svc.InventoryReader.ReadInventory(ctx)
	if err != nil {
		return nil, err
	}

	state := LanDeviceAggregateState{
		Devices:           devices,
		DhcpTags:          []*models.LANCtrlDhcpTagInfo{},
		HostHints:         map[string]HostHintSnapshot{},
		WifiMACs:          map[string]struct{}{},
		TrafficStats:      map[string]TrafficStatSnapshot{},
		StaticAssignments: map[string]*models.LANStaticAssigned{},
		BlockMap:          map[string]*models.LANCtrlSpeedLimitItem{},
		SpeedLimitMap:     map[string]*models.LANCtrlSpeedLimitItem{},
	}

	if dhcpTags, err := svc.DhcpTagReader.ReadDhcpTags(ctx, lanStatus); err == nil {
		state.DhcpTags = dhcpTags
	}
	if hostHints, err := svc.HostHintReader.ReadHostHints(ctx); err == nil {
		state.HostHints = hostHints
	}
	if wifiMACs, err := svc.WifiAssocReader.ReadWifiAssoc(ctx); err == nil {
		state.WifiMACs = wifiMACs
	}

	var lstats *LanStats
	if serviceBackend != nil {
		lstats = serviceBackend.lstats
	}
	if trafficStats, err := svc.TrafficStatReader.ReadTrafficStats(ctx, lstats, state.Devices); err == nil {
		state.TrafficStats = trafficStats
	}

	if staticAssignments, err := svc.StaticAssignmentReader.ReadStaticAssignments(ctx, state.DhcpTags); err == nil {
		state.StaticAssignments = staticAssignments
	}
	if blockMap, speedLimitMap, err := svc.SpeedLimitReader.ReadSpeedLimits(ctx); err == nil {
		state.BlockMap = blockMap
		state.SpeedLimitMap = speedLimitMap
	}

	mergeLanDeviceAggregateState(state)

	return &models.LANDeviceResponse{
		Result: &models.LANDeviceResponseResult{
			Devices:  state.Devices,
			DhcpTags: state.DhcpTags,
		},
	}, nil
}

func mergeLanDeviceAggregateState(state LanDeviceAggregateState) {
	for _, device := range state.Devices {
		if device == nil {
			continue
		}
		mergeLanDeviceHostHint(device, state.HostHints)
		mergeLanDeviceWifiAssoc(device, state.WifiMACs)
		mergeLanDeviceStaticAssignment(device, state.StaticAssignments)
		mergeLanDeviceTrafficStat(device, state.TrafficStats)
		mergeLanDeviceSpeedLimit(device, state.BlockMap, state.SpeedLimitMap)
	}
}

func mergeLanDeviceHostHint(device *models.LANDevice, hostHints map[string]HostHintSnapshot) {
	hint, ok := hostHints[device.Mac]
	if !ok || hint.Hostname == "" {
		return
	}
	device.Hostname = hint.Hostname
}

func mergeLanDeviceWifiAssoc(device *models.LANDevice, wifiMACs map[string]struct{}) {
	if _, ok := wifiMACs[device.Mac]; ok {
		device.Intr = "wifi"
	}
}

func mergeLanDeviceStaticAssignment(device *models.LANDevice, staticAssignments map[string]*models.LANStaticAssigned) {
	val, ok := staticAssignments[device.Mac]
	if !ok || val == nil {
		return
	}
	ensureLanDeviceStaticAssigned(device)

	attached := *val
	attached.AssignedIP = device.StaticAssigned.AssignedIP
	attached.AssignedMac = device.StaticAssigned.AssignedMac
	device.StaticAssigned = &attached
	if attached.Hostname != "" {
		device.Hostname = attached.Hostname
	}
}

func mergeLanDeviceTrafficStat(device *models.LANDevice, trafficStats map[string]TrafficStatSnapshot) {
	val, ok := trafficStats[device.IP]
	if !ok {
		return
	}
	device.UploadSpeed = val.UploadSpeed
	device.DownloadSpeed = val.DownloadSpeed
	device.UploadSpeedStr = utils.ByteCountDecimal(uint64(val.UploadSpeed)) + "/s"
	device.DownloadSpeedStr = utils.ByteCountDecimal(uint64(val.DownloadSpeed)) + "/s"
}

func mergeLanDeviceSpeedLimit(device *models.LANDevice, blockMap, speedLimitMap map[string]*models.LANCtrlSpeedLimitItem) {
	ensureLanDeviceSpeedLimit(device)

	blockItem, blockOk := blockMap[device.Mac]
	speedLimitItem, speedOk := speedLimitMap[device.IP]
	blockOk = blockOk && blockItem != nil
	speedOk = speedOk && speedLimitItem != nil
	if blockOk && speedOk {
		device.SpeedLimit.IP = speedLimitItem.IP
		device.SpeedLimit.Mac = blockItem.Mac
		device.SpeedLimit.UploadSpeed = speedLimitItem.UploadSpeed
		device.SpeedLimit.DownloadSpeed = speedLimitItem.DownloadSpeed
		device.SpeedLimit.Comment = speedLimitItem.Comment
		device.SpeedLimit.NetworkAccess = false
		device.SpeedLimit.Enabled = true
		return
	}
	if speedOk {
		device.SpeedLimit.IP = speedLimitItem.IP
		device.SpeedLimit.UploadSpeed = speedLimitItem.UploadSpeed
		device.SpeedLimit.DownloadSpeed = speedLimitItem.DownloadSpeed
		device.SpeedLimit.Comment = speedLimitItem.Comment
		device.SpeedLimit.NetworkAccess = true
		device.SpeedLimit.Enabled = true
		return
	}
	if blockOk {
		device.SpeedLimit.Mac = blockItem.Mac
		device.SpeedLimit.NetworkAccess = false
		device.SpeedLimit.Enabled = true
	}
}

func ensureLanDeviceStaticAssigned(device *models.LANDevice) {
	if device.StaticAssigned != nil {
		if device.StaticAssigned.AssignedIP == "" {
			device.StaticAssigned.AssignedIP = device.IP
		}
		if device.StaticAssigned.AssignedMac == "" {
			device.StaticAssigned.AssignedMac = device.Mac
		}
		return
	}

	device.StaticAssigned = &models.LANStaticAssigned{
		AssignedIP:  device.IP,
		AssignedMac: device.Mac,
	}
}

func ensureLanDeviceSpeedLimit(device *models.LANDevice) {
	if device.SpeedLimit != nil {
		if device.SpeedLimit.IP == "" {
			device.SpeedLimit.IP = device.IP
		}
		if device.SpeedLimit.Mac == "" {
			device.SpeedLimit.Mac = device.Mac
		}
		return
	}

	device.SpeedLimit = &models.LANCtrlSpeedLimitItem{
		IP:            device.IP,
		Mac:           device.Mac,
		NetworkAccess: true,
	}
}
