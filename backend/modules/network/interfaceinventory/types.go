package interfaceinventory

import "github.com/linkease/quick-start/istore-backend/models"

type Snapshot struct {
	Name     string
	Proto    string
	PortName string
	IPV4Addr string
	IPV6Addr string
}

func ResolveAttachments(allPorts map[string]*models.NetworkPortInfo, deviceName string) ([]*models.NetworkPortInfo, []string) {
	ports := resolveSlavePorts(allPorts, deviceName)
	deviceNames := make([]string, 0, len(ports))
	for _, port := range ports {
		deviceNames = append(deviceNames, port.Name)
	}
	return ports, deviceNames
}

func resolveSlavePorts(allPorts map[string]*models.NetworkPortInfo, deviceName string) []*models.NetworkPortInfo {
	slaves := make([]*models.NetworkPortInfo, 0)
	if len(deviceName) < 1 {
		return slaves
	}
	for _, port := range allPorts {
		if port.Master == deviceName {
			slaves = append(slaves, port)
		}
	}
	if len(slaves) == 0 {
		if port, ok := allPorts[deviceName]; ok {
			slaves = append(slaves, port)
		}
	}
	return slaves
}

func FilterGetConfig(interfaces []*models.NetworkInterfaceInfo) []*models.NetworkInterfaceInfo {
	filtered := make([]*models.NetworkInterfaceInfo, 0, len(interfaces))
	for _, inter := range interfaces {
		if inter.Proto != "dhcpv6" {
			filtered = append(filtered, inter)
		}
	}
	return filtered
}

func BuildStatusResult(interfaces []*models.NetworkInterfaceInfo) *models.NetworkInterfaceStatusResponse {
	return &models.NetworkInterfaceStatusResponse{
		Result: &models.NetworkInterfaceStatusResponseResult{
			Interfaces: interfaces,
		},
	}
}

func BuildGetConfigResult(devices []*models.NetworkPortInfo, interfaces []*models.NetworkInterfaceInfo) *models.NetworkInterfaceGetConfigResponse {
	return &models.NetworkInterfaceGetConfigResponse{
		Result: &models.NetworkInterfaceGetConfigResponseResult{
			Devices:    devices,
			Interfaces: interfaces,
		},
	}
}
