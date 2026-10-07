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
	rateLimit *RateLimitModule
}

type devicePolicyFileSnapshot struct {
	Path   string
	Data   []byte
	Mode   os.FileMode
	Exists bool
}

type systemDevicePolicyBackup struct {
	Kind      string
	Files     []devicePolicyFileSnapshot
	RateLimit *rateLimitPolicyBackup
}

type rateLimitPolicyBackup struct {
	Target rateLimitTarget
	Policy models.DeviceSpeedPolicy
}

func newSystemDevicePolicyStore(inventory *DeviceInventoryModule, rateLimit ...*RateLimitModule) devicePolicyStore {
	store := &systemDevicePolicyStore{inventory: inventory}
	if len(rateLimit) > 0 {
		store.rateLimit = rateLimit[0]
	}
	return store
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
	hasIPv6 := false
	for _, address := range device.Addresses.Current {
		if address != nil && address.Family == 4 {
			currentIPv4 = address.Address
		}
		if address != nil && address.Family == 6 {
			hasIPv6 = true
		}
	}
	policy := &models.DevicePolicy{
		DeviceID: device.DeviceID, DisplayName: device.DisplayName, MAC: device.Mac, CurrentIPv4: currentIPv4,
		Static: &models.DeviceStaticPolicy{}, Speed: &models.DeviceSpeedPolicy{},
		Access: &models.DeviceAccessPolicy{NetworkAccess: true},
		Capabilities: map[string]*models.DevicePolicyCapability{
			"static": {State: "available"}, "speed": {State: "not_installed", Reason: "限速组件未安装"},
			"access": {State: "available"},
		},
	}
	accessCapability := firewallCapabilityAt(deviceRestrictionConfigDir())
	policy.Capabilities["access"] = &models.DevicePolicyCapability{State: models.NormalizeCapabilityState(models.CapabilityState(accessCapability.State)), Reason: accessCapability.Reason}
	if accessCapability.State == "error" {
		policy.Capabilities["access"].Actions = []*models.CapabilityAction{{Kind: "retry"}}
	}
	if device.Mac == "" || currentIPv4 == "" {
		policy.Capabilities["static"] = &models.DevicePolicyCapability{State: "disabled", Reason: "需要当前 IPv4 地址和可识别的 MAC"}
	}

	global, globalErr := NewLanGlobalConfigService().GetGlobalConfigs(ctx)
	if globalErr == nil && global.Result != nil && global.Result.Capabilities != nil && global.Result.Capabilities.SpeedLimit != nil {
		capability := global.Result.Capabilities.SpeedLimit
		policy.Capabilities["speed"] = &models.DevicePolicyCapability{State: capability.State, Reason: capability.Reason, Actions: capability.Actions}
	}
	if device.Mac == "" && policy.Capabilities["speed"].State == "available" {
		policy.Capabilities["speed"] = &models.DevicePolicyCapability{State: "disabled", Reason: "需要可识别的 MAC"}
	}
	if device.Mac == "" && policy.Capabilities["access"].State == "available" {
		policy.Capabilities["access"] = &models.DevicePolicyCapability{State: "disabled", Reason: "需要可识别的 MAC"}
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
	if policy.Capabilities["access"].State == "available" {
		access, readErr := readDeviceAccessPolicyAt(deviceRestrictionConfigDir(), device.Mac)
		if readErr != nil {
			policy.Capabilities["access"] = &models.DevicePolicyCapability{State: "error", Reason: "access_rules_unavailable"}
		} else {
			policy.Access.NetworkAccess = access
		}
	}
	if store.rateLimit != nil {
		speed, enforcement, readErr := store.rateLimit.Inspect(ctx, rateLimitTarget{
			DeviceID: device.DeviceID, MAC: device.Mac, DisplayName: device.DisplayName, CurrentIPv4: currentIPv4,
			ReservedIP: policy.Static.AssignedIP, HasIPv6: hasIPv6,
		})
		if readErr != nil && policy.Capabilities["speed"].State == "available" {
			policy.Capabilities["speed"] = &models.DevicePolicyCapability{State: "error", Reason: "speed_rules_unavailable"}
		} else if readErr == nil {
			policy.RateLimit = enforcement
			if speed != nil {
				policy.Speed = speed
			}
			if policy.Capabilities["speed"].State != "available" {
				policy.Capabilities["speed"].DesiredRetained = speed != nil && speed.Enabled
			}
		}
	} else if currentIPv4 != "" {
		speed, readErr := readDeviceSpeedPolicyAt(deviceRestrictionConfigDir(), currentIPv4)
		if readErr != nil && policy.Capabilities["speed"].State == "available" {
			policy.Capabilities["speed"] = &models.DevicePolicyCapability{State: "error", Reason: "speed_rules_unavailable"}
		} else if readErr == nil && speed != nil && speed.Section != "" {
			policy.Speed = &models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: speed.Upload, DownloadSpeed: speed.Download}
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

func (store *systemDevicePolicyStore) Backup(ctx context.Context, kind string, policies ...*models.DevicePolicy) (devicePolicyBackup, error) {
	paths := []string{"/etc/config/dhcp"}
	if kind == "speed" {
		paths = []string{"/etc/config/eqos"}
		if store.rateLimit != nil && store.rateLimit.provider(ctx) != nil && store.rateLimit.provider(ctx).Name() != "eqos" && len(policies) > 0 && policies[0] != nil {
			current := policies[0]
			reservedIP := ""
			if current.Static != nil {
				reservedIP = current.Static.AssignedIP
			}
			target := rateLimitTarget{DeviceID: current.DeviceID, MAC: current.MAC, DisplayName: current.DisplayName, CurrentIPv4: current.CurrentIPv4, ReservedIP: reservedIP}
			previous := models.DeviceSpeedPolicy{}
			if current.Speed != nil {
				previous = *current.Speed
			}
			return &systemDevicePolicyBackup{Kind: kind, RateLimit: &rateLimitPolicyBackup{Target: target, Policy: previous}}, nil
		}
	}
	if kind == "access" {
		paths = []string{"/etc/config/firewall", immediateAccessIncludePath(deviceRestrictionConfigDir())}
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
	if accumulator := groupBatchAccumulatorFrom(ctx); accumulator != nil && (request.Kind == "access" || request.Kind == "speed") {
		requestCopy := *request
		currentCopy := *current
		accumulator.restrictions = append(accumulator.restrictions, groupBatchRestrictionChange{request: &requestCopy, current: &currentCopy})
		return nil
	}
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
		if store.rateLimit != nil {
			return store.rateLimit.Apply(ctx, rateLimitTarget{
				DeviceID: current.DeviceID, MAC: current.MAC, DisplayName: current.DisplayName, CurrentIPv4: current.CurrentIPv4,
				ReservedIP: current.Static.AssignedIP,
			}, *request.Speed)
		}
		if err := writeDeviceSpeedPolicyAt(deviceRestrictionConfigDir(), current.CurrentIPv4, current.MAC, current.DisplayName, request.Speed.Enabled, request.Speed.UploadSpeed, request.Speed.DownloadSpeed); err != nil {
			return err
		}
		if groupBatchDefersReload(ctx) {
			return nil
		}
		return deviceRestrictionApply(ctx, []string{"eqos"})
	case "access":
		if err := writeDeviceAccessPolicyAt(deviceRestrictionConfigDir(), current.MAC, request.Access.NetworkAccess); err != nil {
			return err
		}
		if groupBatchDefersReload(ctx) {
			return nil
		}
		return deviceAccessRestrictionApply(ctx)
	}
	return errors.New("unsupported policy kind")
}

func (store *systemDevicePolicyStore) Restore(ctx context.Context, raw devicePolicyBackup) error {
	backup, ok := raw.(*systemDevicePolicyBackup)
	if !ok || backup == nil {
		return errors.New("invalid rollback snapshot")
	}
	if backup.RateLimit != nil {
		if store.rateLimit == nil {
			return errors.New("rate limit rollback provider is unavailable")
		}
		return store.rateLimit.Apply(ctx, backup.RateLimit.Target, backup.RateLimit.Policy)
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
	switch backup.Kind {
	case "static":
		commands = append(commands, "/etc/init.d/dnsmasq restart")
	case "access":
		commands = append(commands, "/etc/init.d/firewall restart")
	case "speed":
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
