package service

import (
	"context"
	"net"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/network/interfacewrite"
	"github.com/linkease/quick-start/istore-backend/utils"
)

var networkInterfaceWriteLoadConfig = uci.LoadConfig
var networkInterfaceWriteGetSections = uci.GetSections
var networkInterfaceWriteGetLast = uci.GetLast
var networkInterfaceWriteGet = uci.Get
var networkInterfaceWriteGetConfig = NetworkInterfaceGetConfig
var networkInterfaceWriteBatchRun = utils.BatchRun
var networkInterfaceWriteHasLink = func(name string) bool {
	_, err := net.InterfaceByName(name)
	return err == nil
}

type NetworkInterfaceConfigStore = interfacewrite.Store

type defaultNetworkInterfaceConfigStore struct{}

func NewDefaultNetworkInterfaceConfigStore() NetworkInterfaceConfigStore {
	return &defaultNetworkInterfaceConfigStore{}
}

func (store *defaultNetworkInterfaceConfigStore) ReadSnapshot(ctx context.Context) (NetworkInterfaceConfigSnapshot, error) {
	res, err := networkInterfaceWriteGetConfig(ctx)
	if err != nil {
		return NetworkInterfaceConfigSnapshot{}, err
	}

	networkInterfaceWriteLoadConfig("network", true)
	deviceSecs, _ := networkInterfaceWriteGetSections("network", "device")
	devices := make([]NetworkInterfaceDeviceSnapshot, 0, len(deviceSecs))
	for _, sec := range deviceSecs {
		name, _ := networkInterfaceWriteGetLast("network", sec, "name")
		devices = append(devices, NetworkInterfaceDeviceSnapshot{
			SectionName: sec,
			Name:        name,
		})
	}

	networkInterfaceWriteLoadConfig("firewall", true)
	firewallSecs, _ := networkInterfaceWriteGetSections("firewall", "zone")
	zones := make([]NetworkInterfaceFirewallZoneSnapshot, 0, len(firewallSecs))
	for _, sec := range firewallSecs {
		name, _ := networkInterfaceWriteGetLast("firewall", sec, "name")
		nets, _ := networkInterfaceWriteGet("firewall", sec, "network")
		zones = append(zones, NetworkInterfaceFirewallZoneSnapshot{
			SectionName: sec,
			Name:        name,
			Networks:    nets,
		})
	}

	var interfaces []*models.NetworkInterfaceInfo
	if res != nil && res.Result != nil {
		interfaces = res.Result.Interfaces
	}

	return NetworkInterfaceConfigSnapshot{
		Interfaces:     interfaces,
		DeviceSections: devices,
		FirewallZones:  zones,
	}, nil
}

func (store *defaultNetworkInterfaceConfigStore) NormalizeDevices(name string, devices []string) []string {
	return normalizeNetworkInterfaceDevices(name, devices, networkInterfaceWriteHasLink)
}

func (store *defaultNetworkInterfaceConfigStore) ApplyPlan(ctx context.Context, plan NetworkInterfaceCommandPlan) error {
	for _, batch := range [][]string{plan.DeleteCommands} {
		if len(batch) == 0 {
			continue
		}
		if err := networkInterfaceWriteBatchRun(ctx, batch, 0); err != nil {
			return err
		}
	}
	for _, batch := range plan.BridgeCommands {
		if len(batch) == 0 {
			continue
		}
		if err := networkInterfaceWriteBatchRun(ctx, batch, 0); err != nil {
			return err
		}
	}
	for _, batch := range plan.InterfaceCommands {
		if len(batch) == 0 {
			continue
		}
		if err := networkInterfaceWriteBatchRun(ctx, batch, 0); err != nil {
			return err
		}
	}
	for _, batch := range plan.FirewallCommands {
		if len(batch) == 0 {
			continue
		}
		if err := networkInterfaceWriteBatchRun(ctx, batch, 0); err != nil {
			return err
		}
	}
	return nil
}
