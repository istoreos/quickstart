package service

import (
	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/network/interfaceinventory"
)

type NetworkInterfaceInventorySnapshot = interfaceinventory.Snapshot

func resolveNetworkInterfaceAttachments(allPorts map[string]*models.NetworkPortInfo, deviceName string) ([]*models.NetworkPortInfo, []string) {
	return interfaceinventory.ResolveAttachments(allPorts, deviceName)
}

func filterNetworkInterfaceGetConfig(interfaces []*models.NetworkInterfaceInfo) []*models.NetworkInterfaceInfo {
	return interfaceinventory.FilterGetConfig(interfaces)
}

func buildNetworkInterfaceStatusResult(interfaces []*models.NetworkInterfaceInfo) *models.NetworkInterfaceStatusResponse {
	return interfaceinventory.BuildStatusResult(interfaces)
}

func buildNetworkInterfaceGetConfigResult(devices []*models.NetworkPortInfo, interfaces []*models.NetworkInterfaceInfo) *models.NetworkInterfaceGetConfigResponse {
	return interfaceinventory.BuildGetConfigResult(devices, interfaces)
}
