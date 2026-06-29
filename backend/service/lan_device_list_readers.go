package service

import (
	"context"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
)

type DhcpTagReader interface {
	ReadDhcpTags(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error)
}

type HostHintReader interface {
	ReadHostHints(ctx context.Context) (map[string]HostHintSnapshot, error)
}

type WifiAssocReader interface {
	ReadWifiAssoc(ctx context.Context) (map[string]struct{}, error)
}

type TrafficStatReader interface {
	ReadTrafficStats(ctx context.Context, lstats *LanStats, devices models.LANDevices) (map[string]TrafficStatSnapshot, error)
}

type StaticAssignmentReader interface {
	ReadStaticAssignments(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) (map[string]*models.LANStaticAssigned, error)
}

type LanDeviceSpeedLimitReader interface {
	ReadSpeedLimits(ctx context.Context) (map[string]*models.LANCtrlSpeedLimitItem, map[string]*models.LANCtrlSpeedLimitItem, error)
}

type defaultDhcpTagReader struct{}
type defaultHostHintReader struct{}
type defaultWifiAssocReader struct{}
type defaultTrafficStatReader struct{}
type defaultStaticAssignmentReader struct {
	listReader StaticAssignmentListReader
}
type defaultLanDeviceSpeedLimitReader struct {
	store LanSpeedLimitRuleStore
}

var _ DhcpTagReader = (*defaultDhcpTagReader)(nil)
var _ HostHintReader = (*defaultHostHintReader)(nil)
var _ WifiAssocReader = (*defaultWifiAssocReader)(nil)
var _ TrafficStatReader = (*defaultTrafficStatReader)(nil)
var _ StaticAssignmentReader = (*defaultStaticAssignmentReader)(nil)
var _ LanDeviceSpeedLimitReader = (*defaultLanDeviceSpeedLimitReader)(nil)

var lanDeviceListLoadConfig = uci.LoadConfig
var newDefaultStaticAssignmentListReader = NewDefaultStaticAssignmentListReader

func NewDefaultDhcpTagReader() DhcpTagReader {
	return &defaultDhcpTagReader{}
}

func NewDefaultHostHintReader() HostHintReader {
	return &defaultHostHintReader{}
}

func NewDefaultWifiAssocReader() WifiAssocReader {
	return &defaultWifiAssocReader{}
}

func NewDefaultTrafficStatReader() TrafficStatReader {
	return &defaultTrafficStatReader{}
}

func NewDefaultStaticAssignmentReader() StaticAssignmentReader {
	return &defaultStaticAssignmentReader{
		listReader: newDefaultStaticAssignmentListReader(),
	}
}

func NewDefaultLanDeviceSpeedLimitReader() LanDeviceSpeedLimitReader {
	return &defaultLanDeviceSpeedLimitReader{
		store: NewDefaultLanSpeedLimitRuleStore(),
	}
}

func buildTrafficStatSnapshot(upload, download int64) TrafficStatSnapshot {
	return TrafficStatSnapshot{
		UploadSpeed:   upload,
		DownloadSpeed: download,
	}
}

func buildHostHintHostname(name string) string {
	match := matchStringOnce(name, `^(.+)\.`)
	if match == nil {
		return ""
	}
	return match[1]
}

func preloadStaticAssignmentConfigs() {
	lanDeviceListLoadConfig("dhcp", true)
	lanDeviceListLoadConfig("floatip", true)
}

func preloadSpeedLimitConfigs() {
	preloadLanSpeedLimitRuleConfigs()
}

func buildBestEffortHostHints(raw ubusHostHintMap, err error) (map[string]HostHintSnapshot, error) {
	if err != nil {
		return map[string]HostHintSnapshot{}, nil
	}

	hints := make(map[string]HostHintSnapshot, len(raw))
	for mac, hint := range raw {
		if hint == nil {
			continue
		}
		hints[strings.ToUpper(strings.TrimSpace(mac))] = HostHintSnapshot{
			Hostname: buildHostHintHostname(hint.Name),
		}
	}
	return hints, nil
}

func buildBestEffortStaticAssignments(items []*models.LANStaticAssigned, err error) (map[string]*models.LANStaticAssigned, error) {
	if err != nil {
		return map[string]*models.LANStaticAssigned{}, nil
	}

	assigned := make(map[string]*models.LANStaticAssigned, len(items))
	for _, item := range items {
		if item == nil || item.AssignedMac == "" {
			continue
		}
		assigned[item.AssignedMac] = item
	}
	return assigned, nil
}

func buildBestEffortSpeedLimitMaps(blocks, speedLimits []*models.LANCtrlSpeedLimitItem, err error) (map[string]*models.LANCtrlSpeedLimitItem, map[string]*models.LANCtrlSpeedLimitItem, error) {
	if err != nil {
		return map[string]*models.LANCtrlSpeedLimitItem{}, map[string]*models.LANCtrlSpeedLimitItem{}, nil
	}

	blockMap := make(map[string]*models.LANCtrlSpeedLimitItem, len(blocks))
	speedMap := make(map[string]*models.LANCtrlSpeedLimitItem, len(speedLimits))
	for _, item := range blocks {
		if item == nil || item.Mac == "" {
			continue
		}
		blockMap[item.Mac] = item
	}
	for _, item := range speedLimits {
		if item == nil || item.IP == "" {
			continue
		}
		speedMap[item.IP] = item
	}

	return blockMap, speedMap, nil
}

func (reader *defaultDhcpTagReader) ReadDhcpTags(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error) {
	tagList, _, _ := getDhcpOfLAN(ctx, &ubusLanStatus{
		lanAddr:          lanStatus.LanAddr,
		nexthop:          lanStatus.Nexthop,
		isDefaultGateway: lanStatus.IsDefaultGateway,
	})
	return tagList, nil
}

func (reader *defaultHostHintReader) ReadHostHints(ctx context.Context) (map[string]HostHintSnapshot, error) {
	raw := ubusHostHintMap{}
	return buildBestEffortHostHints(raw, UbusCallWithObject(ctx, "luci-rpc getHostHints", &raw))
}

func (reader *defaultWifiAssocReader) ReadWifiAssoc(ctx context.Context) (map[string]struct{}, error) {
	drive := wifiSelect()
	if drive == nil {
		return map[string]struct{}{}, nil
	}
	macs, err := drive.AssocMacList(ctx)
	if err != nil {
		return map[string]struct{}{}, nil
	}
	return macs, nil
}

func (reader *defaultTrafficStatReader) ReadTrafficStats(ctx context.Context, lstats *LanStats, devices models.LANDevices) (map[string]TrafficStatSnapshot, error) {
	_ = ctx
	stats := make(map[string]TrafficStatSnapshot, len(devices))
	if lstats == nil {
		return stats, nil
	}

	for _, dev := range devices {
		if dev == nil || dev.IP == "" {
			continue
		}
		speedData := lstats.reqHosts(dev.IP, true)
		if len(speedData) == 0 || len(speedData[0].items) == 0 {
			continue
		}
		item := speedData[0].items[0]
		stats[dev.IP] = buildTrafficStatSnapshot(item.txAvg, item.rxAvg)
	}
	return stats, nil
}

func (reader *defaultStaticAssignmentReader) ReadStaticAssignments(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) (map[string]*models.LANStaticAssigned, error) {
	items, err := reader.listReader.ReadStaticAssignments(ctx, tagList)
	return buildBestEffortStaticAssignments(items, err)
}

func (reader *defaultLanDeviceSpeedLimitReader) ReadSpeedLimits(ctx context.Context) (map[string]*models.LANCtrlSpeedLimitItem, map[string]*models.LANCtrlSpeedLimitItem, error) {
	return buildBestEffortSpeedLimitMaps(reader.store.ReadRuleLists(ctx))
}
