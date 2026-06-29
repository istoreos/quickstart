package service

import (
	"context"
	"errors"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/network/interfaceinventory"
)

type NetworkInterfaceInventoryReader = interfaceinventory.InventoryReader
type NetworkInterfaceFirewallBindingReader = interfaceinventory.FirewallBindingReader
type NetworkInterfacePortAttachmentResolver = interfaceinventory.PortAttachmentResolver

var readNetworkInterfaceInventoryUbusCall = UbusCall

var readNetworkInterfaceInventorySnapshots = func(ctx context.Context) ([]NetworkInterfaceInventorySnapshot, error) {
	interfaceDump, err := readNetworkInterfaceInventoryUbusCall(ctx, "network.interface dump")
	if err != nil {
		return nil, err
	}
	if interfaceDump == nil {
		return nil, errors.New("network interface dump is empty")
	}

	interfaces := interfaceDump.Get("interface")
	snapshots := make([]NetworkInterfaceInventorySnapshot, 0, jsonArrayLen(interfaces))
	for i, ifaceCount := 0, jsonArrayLen(interfaces); i < ifaceCount; i++ {
		iface := interfaces.GetIndex(i)
		snapshots = append(snapshots, NetworkInterfaceInventorySnapshot{
			Name:     iface.Get("interface").MustString(),
			Proto:    iface.Get("proto").MustString(),
			PortName: iface.Get("device").MustString(),
			IPV4Addr: iface.Get("ipv4-address").GetIndex(0).Get("address").MustString(),
			IPV6Addr: iface.Get("ipv6-address").GetIndex(0).Get("address").MustString(),
		})
	}
	return snapshots, nil
}

var readNetworkInterfaceFirewallLoadConfig = uci.LoadConfig
var readNetworkInterfaceFirewallGetSections = uci.GetSections
var readNetworkInterfaceFirewallGetLast = uci.GetLast
var readNetworkInterfaceFirewallGet = uci.Get

var readNetworkInterfaceFirewallBindings = func(ctx context.Context) (map[string]string, error) {
	if err := readNetworkInterfaceFirewallLoadConfig("firewall", true); err != nil {
		return nil, err
	}

	sections, _ := readNetworkInterfaceFirewallGetSections("firewall", "zone")
	bindings := make(map[string]string)
	for _, section := range sections {
		name, _ := readNetworkInterfaceFirewallGetLast("firewall", section, "name")
		networks, _ := readNetworkInterfaceFirewallGet("firewall", section, "network")
		for _, network := range networks {
			bindings[network] = name
		}
	}
	return bindings, nil
}

type defaultNetworkInterfaceInventoryReader struct{}

func newDefaultNetworkInterfaceInventoryReader() NetworkInterfaceInventoryReader {
	return &defaultNetworkInterfaceInventoryReader{}
}

func (reader *defaultNetworkInterfaceInventoryReader) Read(ctx context.Context) ([]NetworkInterfaceInventorySnapshot, error) {
	snapshots, err := readNetworkInterfaceInventorySnapshots(ctx)
	if err != nil {
		return nil, errors.New("获取网络接口失败")
	}

	filtered := make([]NetworkInterfaceInventorySnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.Name == "docker" || snapshot.Name == "loopback" {
			continue
		}
		filtered = append(filtered, snapshot)
	}
	return filtered, nil
}

type defaultNetworkInterfaceFirewallBindingReader struct{}

func newDefaultNetworkInterfaceFirewallBindingReader() NetworkInterfaceFirewallBindingReader {
	return &defaultNetworkInterfaceFirewallBindingReader{}
}

func (reader *defaultNetworkInterfaceFirewallBindingReader) Read(ctx context.Context) (map[string]string, error) {
	return readNetworkInterfaceFirewallBindings(ctx)
}

type defaultNetworkInterfacePortAttachmentResolver struct{}

func newDefaultNetworkInterfacePortAttachmentResolver() NetworkInterfacePortAttachmentResolver {
	return &defaultNetworkInterfacePortAttachmentResolver{}
}

func (resolver *defaultNetworkInterfacePortAttachmentResolver) Resolve(allPorts map[string]*models.NetworkPortInfo, deviceName string) ([]*models.NetworkPortInfo, []string) {
	return resolveNetworkInterfaceAttachments(allPorts, deviceName)
}
