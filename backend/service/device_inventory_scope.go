package service

import (
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/istoreos/quickstart/backend/models"
)

const unassignedLAN = "@unassigned" // Cannot collide with a valid UCI network name.

func inventoryLANID(network string) string {
	if network == "" {
		return defaultLAN
	}
	return network
}

func storedLANID(network string) string {
	if network == "" {
		return unassignedLAN
	}
	return network
}

func wireLANID(network string) string {
	if network == unassignedLAN {
		return ""
	}
	return network
}

func inventoryLANKey(network, identity string) string {
	network = storedLANID(network)
	if network == defaultLAN {
		return identity
	}
	return network + "\x00" + identity
}

func lanStatePath(path, network string) string {
	if network == "" || network == defaultLAN {
		return path
	}
	return filepath.Join(filepath.Dir(path), "networks", network, filepath.Base(path))
}

func lanStateFiles(path string) map[string]string {
	files := map[string]string{defaultLAN: path}
	paths, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "networks", "*", filepath.Base(path)))
	for _, candidate := range paths {
		network := filepath.Base(filepath.Dir(candidate))
		if network != unassignedLAN && !validRouterContextLAN.MatchString(network) {
			continue
		}
		info, err := os.Lstat(candidate)
		if err == nil && info.Mode().IsRegular() {
			files[network] = candidate
		}
	}
	return files
}

func inventoryLANMembership(network string, online bool) string {
	if network == unassignedLAN {
		return "unconfirmed"
	}
	if online {
		return "observed"
	}
	return "historical"
}

func markLANAddressZones(addresses []*models.DeviceInventoryAddress, device string) {
	for _, address := range addresses {
		parsed, err := netip.ParseAddr(address.Address)
		if err == nil && parsed.Is6() && parsed.IsLinkLocalUnicast() {
			address.Zone = device
		}
	}
}

func inventoryNetworkHealth(source deviceInventorySourceSnapshot) map[string]*models.DeviceInventoryHealth {
	health := map[string]*models.DeviceInventoryHealth{}
	for network, reasons := range source.NetworkReasons {
		health[wireLANID(network)] = inventoryHealth(reasons)
	}
	if health[defaultLAN] == nil {
		health[defaultLAN] = inventoryHealth(source.Reasons)
	}
	return health
}

func addInventoryNetworkReason(source *deviceInventorySourceSnapshot, network, reason string) {
	if source.NetworkReasons == nil {
		source.NetworkReasons = map[string][]string{defaultLAN: append([]string{}, source.Reasons...)}
	}
	source.NetworkReasons[network] = append(source.NetworkReasons[network], reason)
	source.Reasons = append(source.Reasons, reason)
}

func addInventoryGlobalReason(source *deviceInventorySourceSnapshot, reason string) {
	if source.NetworkReasons == nil {
		source.NetworkReasons = map[string][]string{defaultLAN: append([]string{}, source.Reasons...)}
	}
	for network := range source.NetworkReasons {
		source.NetworkReasons[network] = append(source.NetworkReasons[network], reason)
	}
	source.Reasons = append(source.Reasons, reason)
}

func inventoryHealth(reasons []string) *models.DeviceInventoryHealth {
	reasons = uniqueSortedStrings(reasons)
	state := "ready"
	if len(reasons) > 0 {
		state = "partial"
	}
	return &models.DeviceInventoryHealth{State: state, Reasons: reasons}
}

func filterLANInventory(response *models.DeviceInventoryResponse, network string) *models.DeviceInventoryResponse {
	if response == nil || response.Result == nil || network == "all" {
		return response
	}
	devices := []*models.DeviceInventoryItem{}
	for _, device := range response.Result.Devices {
		if device.NetworkID == network {
			devices = append(devices, device)
		}
	}
	health := response.Result.NetworkHealth[network]
	if health == nil {
		health = response.Result.Health
	}
	if len(devices) == len(response.Result.Devices) && health.State == response.Result.Health.State && strings.Join(health.Reasons, "|") == strings.Join(response.Result.Health.Reasons, "|") {
		return response
	}
	return &models.DeviceInventoryResponse{Result: &models.DeviceInventoryResult{Devices: devices, Health: health}}
}
