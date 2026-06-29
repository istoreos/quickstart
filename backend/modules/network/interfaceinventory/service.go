package interfaceinventory

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

type PortStatusReader interface {
	Read(ctx context.Context) ([]*models.NetworkPortInfo, error)
}

type InventoryReader interface {
	Read(ctx context.Context) ([]Snapshot, error)
}

type FirewallBindingReader interface {
	Read(ctx context.Context) (map[string]string, error)
}

type PortAttachmentResolver interface {
	Resolve(allPorts map[string]*models.NetworkPortInfo, deviceName string) ([]*models.NetworkPortInfo, []string)
}

type Service struct {
	portStatusReader       PortStatusReader
	inventoryReader        InventoryReader
	firewallBindingReader  FirewallBindingReader
	portAttachmentResolver PortAttachmentResolver
}

func NewService(
	portStatusReader PortStatusReader,
	inventoryReader InventoryReader,
	firewallBindingReader FirewallBindingReader,
	portAttachmentResolver PortAttachmentResolver,
) *Service {
	return &Service{
		portStatusReader:       portStatusReader,
		inventoryReader:        inventoryReader,
		firewallBindingReader:  firewallBindingReader,
		portAttachmentResolver: portAttachmentResolver,
	}
}

func (svc *Service) ListInventory(ctx context.Context) ([]*models.NetworkInterfaceInfo, error) {
	ports, err := svc.portStatusReader.Read(ctx)
	if err != nil {
		return nil, err
	}
	allPorts := make(map[string]*models.NetworkPortInfo, len(ports))
	for _, port := range ports {
		allPorts[port.Name] = port
	}

	snapshots, err := svc.inventoryReader.Read(ctx)
	if err != nil {
		return nil, err
	}

	bindings, err := svc.firewallBindingReader.Read(ctx)
	if err != nil {
		return nil, err
	}

	interfaces := make([]*models.NetworkInterfaceInfo, 0, len(snapshots))
	for _, snapshot := range snapshots {
		resolvedPorts, deviceNames := svc.portAttachmentResolver.Resolve(allPorts, snapshot.PortName)
		interfaces = append(interfaces, &models.NetworkInterfaceInfo{
			Name:         snapshot.Name,
			Proto:        snapshot.Proto,
			IPV4Addr:     snapshot.IPV4Addr,
			IPV6Addr:     snapshot.IPV6Addr,
			PortName:     snapshot.PortName,
			Ports:        resolvedPorts,
			DeviceNames:  deviceNames,
			FirewallType: bindings[snapshot.Name],
		})
	}
	return interfaces, nil
}
