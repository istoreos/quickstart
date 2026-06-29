package service

import (
	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/network/interfacewrite"
)

type NetworkInterfaceWriteInput = interfacewrite.Input

type NetworkInterfaceConfigSnapshot = interfacewrite.Snapshot

type NetworkInterfaceDeviceSnapshot = interfacewrite.DeviceSnapshot

type NetworkInterfaceFirewallZoneSnapshot = interfacewrite.FirewallZoneSnapshot

type NetworkInterfaceWritePlan = interfacewrite.WritePlan

type NetworkInterfaceCommandPlan = interfacewrite.CommandPlan

func buildNetworkInterfaceDeleteCommands(deleteInterfaces []string) []string {
	return interfacewrite.BuildDeleteCommands(deleteInterfaces)
}

func buildNetworkInterfaceDeletePlan(existing []*models.NetworkInterfaceInfo, configs []*models.NetworkInterfaceConfig) []string {
	return interfacewrite.BuildDeletePlan(existing, configs)
}

func normalizeNetworkInterfaceDevices(name string, devices []string, hasPort func(string) bool) []string {
	return interfacewrite.NormalizeDevices(name, devices, hasPort)
}

func buildNetworkInterfaceBridgePlan(cfg *models.NetworkInterfaceConfig, devices []string, sections []NetworkInterfaceDeviceSnapshot) []string {
	return interfacewrite.BuildBridgePlan(cfg, devices, sections)
}

func buildNetworkInterfaceInterfacePlan(cfg *models.NetworkInterfaceConfig, deviceName string, bridged bool) []string {
	return interfacewrite.BuildInterfacePlan(cfg, deviceName, bridged)
}

func buildNetworkInterfaceFirewallBindingPlan(cfg *models.NetworkInterfaceConfig, zones []NetworkInterfaceFirewallZoneSnapshot) []string {
	return interfacewrite.BuildFirewallBindingPlan(cfg, zones)
}
