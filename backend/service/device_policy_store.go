package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/istoreos/quickstart/backend/models"
	"github.com/istoreos/quickstart/backend/utils"
)

type systemDevicePolicyStore struct {
	inventory *DeviceInventoryModule
}

type devicePolicyFileSnapshot struct {
	Path   string
	Data   []byte
	Mode   os.FileMode
	Exists bool
}

type systemDevicePolicyBackup struct {
	Kind  string
	Files []devicePolicyFileSnapshot
}

func newSystemDevicePolicyStore(inventory *DeviceInventoryModule) devicePolicyStore {
	return &systemDevicePolicyStore{inventory: inventory}
}

func (store *systemDevicePolicyStore) Get(ctx context.Context, deviceID string) (*models.DevicePolicy, error) {
	if store.inventory == nil {
		return nil, errors.New("device inventory is unavailable")
	}
	inventory, err := store.inventory.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	var device *models.DeviceInventoryItem
	for _, item := range inventory.Result.Devices {
		if item != nil && item.DeviceID == deviceID {
			device = item
			break
		}
	}
	if device == nil {
		return nil, errors.New("device was not found")
	}
	currentIPv4 := ""
	for _, address := range device.Addresses.Current {
		if address != nil && address.Family == 4 {
			currentIPv4 = address.Address
			break
		}
	}
	policy := &models.DevicePolicy{
		DeviceID: device.DeviceID, DisplayName: device.DisplayName, MAC: device.Mac, CurrentIPv4: currentIPv4,
		Static: &models.DeviceStaticPolicy{}, Speed: &models.DeviceSpeedPolicy{},
		Access: &models.DeviceAccessPolicy{NetworkAccess: true},
		Capabilities: map[string]*models.DevicePolicyCapability{
			"static": {State: "available"}, "speed": {State: "not_installed", Reason: "限速组件未安装"},
			"access": {State: "not_installed", Reason: "限速组件未安装"},
		},
	}
	if device.Mac == "" || currentIPv4 == "" {
		policy.Capabilities["static"] = &models.DevicePolicyCapability{State: "disabled", Reason: "需要当前 IPv4 地址和可识别的 MAC"}
	}

	global, globalErr := NewLanGlobalConfigService().GetGlobalConfigs(ctx)
	if globalErr == nil && global.Result != nil && global.Result.Capabilities != nil && global.Result.Capabilities.SpeedLimit != nil {
		capability := global.Result.Capabilities.SpeedLimit
		policy.Capabilities["speed"] = &models.DevicePolicyCapability{State: capability.State, Reason: capability.Reason}
		policy.Capabilities["access"] = &models.DevicePolicyCapability{State: capability.State, Reason: capability.Reason}
	}
	if device.Mac == "" || currentIPv4 == "" {
		for _, kind := range []string{"speed", "access"} {
			if policy.Capabilities[kind].State == "available" {
				policy.Capabilities[kind] = &models.DevicePolicyCapability{State: "disabled", Reason: "需要当前 IPv4 地址和可识别的 MAC"}
			}
		}
	}

	staticRules, staticErr := NewLanStaticDeviceListService().GetListStaticDevices(ctx)
	if staticErr == nil {
		for _, rule := range staticRules.Result {
			if rule != nil && normalizeInventoryMAC(rule.AssignedMac) == normalizeInventoryMAC(device.Mac) {
				policy.Static = &models.DeviceStaticPolicy{
					Enabled: true, AssignedIP: rule.AssignedIP, BindIP: rule.BindIP, Hostname: rule.Hostname,
					TagName: rule.TagName, TagTitle: rule.TagTitle,
				}
				break
			}
		}
	}
	if policy.Capabilities["speed"].State == "available" {
		speedRules, speedErr := NewLanSpeedLimitedDeviceListService().GetListSpeedLimitedDevices(ctx)
		if speedErr != nil {
			policy.Capabilities["speed"] = &models.DevicePolicyCapability{State: "error", Reason: "读取限速规则失败"}
			policy.Capabilities["access"] = &models.DevicePolicyCapability{State: "error", Reason: "读取联网规则失败"}
		} else {
			for _, rule := range speedRules.Result {
				if rule == nil || (normalizeInventoryMAC(rule.Mac) != normalizeInventoryMAC(device.Mac) && rule.IP != currentIPv4) {
					continue
				}
				policy.Access.NetworkAccess = rule.NetworkAccess
				if rule.NetworkAccess && (rule.UploadSpeed > 0 || rule.DownloadSpeed > 0) {
					policy.Speed = &models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: rule.UploadSpeed, DownloadSpeed: rule.DownloadSpeed}
				}
			}
		}
	}
	return policy, nil
}

func (store *systemDevicePolicyStore) ListRules(ctx context.Context) (*models.DevicePolicyRulesResult, error) {
	result := &models.DevicePolicyRulesResult{Static: []*models.LANStaticAssigned{}, Speed: []*models.LANCtrlSpeedLimitItem{}}
	staticRules, err := NewLanStaticDeviceListService().GetListStaticDevices(ctx)
	if err != nil {
		return nil, err
	}
	result.Static = staticRules.Result
	speedRules, err := NewLanSpeedLimitedDeviceListService().GetListSpeedLimitedDevices(ctx)
	if err == nil {
		result.Speed = speedRules.Result
	}
	return result, nil
}

func (store *systemDevicePolicyStore) Backup(_ context.Context, kind string) (devicePolicyBackup, error) {
	paths := []string{"/etc/config/dhcp"}
	if kind == "speed" || kind == "access" {
		paths = []string{"/etc/config/eqos", "/etc/config/firewall"}
	}
	backup := &systemDevicePolicyBackup{Kind: kind, Files: make([]devicePolicyFileSnapshot, 0, len(paths))}
	for _, path := range paths {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			backup.Files = append(backup.Files, devicePolicyFileSnapshot{Path: path})
			continue
		}
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		backup.Files = append(backup.Files, devicePolicyFileSnapshot{Path: path, Data: data, Mode: info.Mode(), Exists: true})
	}
	return backup, nil
}

func (store *systemDevicePolicyStore) Apply(ctx context.Context, request *models.DevicePolicyApplyRequest, current *models.DevicePolicy) error {
	switch request.Kind {
	case "static":
		action := "add"
		if current.Static.Enabled {
			action = "modify"
		}
		if !request.Static.Enabled {
			action = "delete"
		}
		return NewDefaultLanStaticAssignmentWriteService().ApplyStaticAssignment(ctx, StaticAssignmentWriteInput{
			Action: action, AssignedMAC: current.MAC, AssignedIP: request.Static.AssignedIP,
			BindIP: request.Static.BindIP, Hostname: request.Static.Hostname, TagName: request.Static.TagName, TagTitle: request.Static.TagTitle,
		})
	case "speed":
		action := "modify"
		if !request.Speed.Enabled {
			action = "delete"
		}
		return NewDefaultLanSpeedLimitWriteService().UpsertSpeedLimitRule(ctx, SpeedLimitWriteInput{
			Action: action, IP: current.CurrentIPv4, MAC: current.MAC, NetworkAccess: true,
			UploadSpeed: request.Speed.UploadSpeed, DownloadSpeed: request.Speed.DownloadSpeed, Comment: current.DisplayName,
		})
	case "access":
		action := "modify"
		upload, download := current.Speed.UploadSpeed, current.Speed.DownloadSpeed
		if request.Access.NetworkAccess && !current.Speed.Enabled {
			action = "delete"
		}
		return NewDefaultLanSpeedLimitWriteService().UpsertSpeedLimitRule(ctx, SpeedLimitWriteInput{
			Action: action, IP: current.CurrentIPv4, MAC: current.MAC, NetworkAccess: request.Access.NetworkAccess,
			UploadSpeed: upload, DownloadSpeed: download, Comment: current.DisplayName,
		})
	}
	return errors.New("unsupported policy kind")
}

func (store *systemDevicePolicyStore) Restore(ctx context.Context, raw devicePolicyBackup) error {
	backup, ok := raw.(*systemDevicePolicyBackup)
	if !ok || backup == nil {
		return errors.New("invalid rollback snapshot")
	}
	for _, file := range backup.Files {
		if !file.Exists {
			if err := os.Remove(file.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			continue
		}
		tmp := filepath.Join(filepath.Dir(file.Path), ".quickstart-"+filepath.Base(file.Path)+".rollback")
		if err := os.WriteFile(tmp, file.Data, file.Mode.Perm()); err != nil {
			return err
		}
		if err := os.Rename(tmp, file.Path); err != nil {
			return err
		}
	}
	commands := []string{"uci reload_config"}
	if backup.Kind == "static" {
		commands = append(commands, "/etc/init.d/dnsmasq restart")
	} else {
		commands = append(commands, "/etc/init.d/firewall reload")
		if _, err := os.Stat("/etc/init.d/eqos"); err == nil {
			commands = append(commands, "/etc/init.d/eqos restart")
		}
	}
	if err := utils.BatchRun(ctx, commands, 0); err != nil {
		return fmt.Errorf("restore policy services: %w", err)
	}
	return nil
}

func policyCapabilityReason(capability *models.DevicePolicyCapability) string {
	if capability == nil {
		return ""
	}
	return strings.TrimSpace(capability.Reason)
}
