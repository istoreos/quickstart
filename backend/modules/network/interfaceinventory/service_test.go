package interfaceinventory

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakePortStatusReader struct {
	ports []*models.NetworkPortInfo
	err   error
}

func (reader *fakePortStatusReader) Read(ctx context.Context) ([]*models.NetworkPortInfo, error) {
	return reader.ports, reader.err
}

type fakeInventoryReader struct {
	snapshots []Snapshot
	err       error
}

func (reader *fakeInventoryReader) Read(ctx context.Context) ([]Snapshot, error) {
	return reader.snapshots, reader.err
}

type fakeFirewallBindingReader struct {
	bindings map[string]string
	err      error
}

func (reader *fakeFirewallBindingReader) Read(ctx context.Context) (map[string]string, error) {
	return reader.bindings, reader.err
}

type fakeAttachmentResolver struct {
	resolve func(allPorts map[string]*models.NetworkPortInfo, deviceName string) ([]*models.NetworkPortInfo, []string)
}

func (resolver *fakeAttachmentResolver) Resolve(allPorts map[string]*models.NetworkPortInfo, deviceName string) ([]*models.NetworkPortInfo, []string) {
	return resolver.resolve(allPorts, deviceName)
}

func TestServiceBuildsInventoryWithPortsDeviceNamesAndFirewallType(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&fakePortStatusReader{
			ports: []*models.NetworkPortInfo{
				{Name: "eth0"},
				{Name: "lan1", Master: "br-lan"},
			},
		},
		&fakeInventoryReader{
			snapshots: []Snapshot{
				{Name: "wan", Proto: "dhcp", PortName: "eth0", IPV4Addr: "192.0.2.2"},
				{Name: "lan", Proto: "static", PortName: "br-lan", IPV4Addr: "192.168.100.1"},
				{Name: "wan6", Proto: "dhcpv6", PortName: "eth0", IPV6Addr: "2001:db8::2"},
			},
		},
		&fakeFirewallBindingReader{
			bindings: map[string]string{"wan": "wan", "lan": "lan", "wan6": "wan"},
		},
		&fakeAttachmentResolver{resolve: ResolveAttachments},
	)

	interfaces, err := svc.ListInventory(context.Background())
	if err != nil {
		t.Fatalf("unexpected inventory service error: %v", err)
	}
	if len(interfaces) != 3 {
		t.Fatalf("expected full inventory including dhcpv6 entry, got %#v", interfaces)
	}
	if interfaces[0].Name != "wan" || interfaces[0].FirewallType != "wan" || len(interfaces[0].Ports) != 1 || interfaces[0].Ports[0].Name != "eth0" {
		t.Fatalf("unexpected wan inventory record: %#v", interfaces[0])
	}
	if interfaces[1].Name != "lan" || len(interfaces[1].DeviceNames) != 1 || interfaces[1].DeviceNames[0] != "lan1" {
		t.Fatalf("unexpected lan inventory record: %#v", interfaces[1])
	}
	if interfaces[2].Proto != "dhcpv6" {
		t.Fatalf("expected shared inventory service not to filter dhcpv6, got %#v", interfaces[2])
	}
}

func TestServicePropagatesReaderErrors(t *testing.T) {
	t.Parallel()

	portErr := errors.New("port status failed")
	svc := NewService(&fakePortStatusReader{err: portErr}, &fakeInventoryReader{}, &fakeFirewallBindingReader{}, &fakeAttachmentResolver{resolve: ResolveAttachments})
	if _, err := svc.ListInventory(context.Background()); !errors.Is(err, portErr) {
		t.Fatalf("expected port status error, got %v", err)
	}

	inventoryErr := errors.New("inventory failed")
	svc = NewService(
		&fakePortStatusReader{ports: []*models.NetworkPortInfo{{Name: "eth0"}}},
		&fakeInventoryReader{err: inventoryErr},
		&fakeFirewallBindingReader{},
		&fakeAttachmentResolver{resolve: ResolveAttachments},
	)
	if _, err := svc.ListInventory(context.Background()); !errors.Is(err, inventoryErr) {
		t.Fatalf("expected inventory reader error, got %v", err)
	}

	firewallErr := errors.New("firewall failed")
	svc = NewService(
		&fakePortStatusReader{ports: []*models.NetworkPortInfo{{Name: "eth0"}}},
		&fakeInventoryReader{snapshots: []Snapshot{{Name: "wan", Proto: "dhcp", PortName: "eth0"}}},
		&fakeFirewallBindingReader{err: firewallErr},
		&fakeAttachmentResolver{resolve: ResolveAttachments},
	)
	if _, err := svc.ListInventory(context.Background()); !errors.Is(err, firewallErr) {
		t.Fatalf("expected firewall binding error, got %v", err)
	}
}
