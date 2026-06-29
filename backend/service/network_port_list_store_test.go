package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	simplejson "github.com/bitly/go-simplejson"
	"github.com/linkease/quick-start/istore-backend/models"
)

var networkPortListReaderTestMu sync.Mutex

func TestDefaultNetworkPortStatusReaderDelegatesToPortStatusSeam(t *testing.T) {
	networkPortListReaderTestMu.Lock()
	defer networkPortListReaderTestMu.Unlock()

	original := readNetworkPortStatus
	readNetworkPortStatus = func(ctx context.Context) ([]*models.NetworkPortInfo, error) {
		return []*models.NetworkPortInfo{{Name: "eth0"}}, nil
	}
	defer func() { readNetworkPortStatus = original }()

	reader := newDefaultNetworkPortStatusReader()
	ports, err := reader.Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected reader error: %v", err)
	}
	if len(ports) != 1 || ports[0].Name != "eth0" {
		t.Fatalf("unexpected ports from status reader: %#v", ports)
	}
}

func TestDefaultNetworkPortStatusReaderPropagatesError(t *testing.T) {
	networkPortListReaderTestMu.Lock()
	defer networkPortListReaderTestMu.Unlock()

	readErr := errors.New("port status failed")
	original := readNetworkPortStatus
	readNetworkPortStatus = func(ctx context.Context) ([]*models.NetworkPortInfo, error) {
		return nil, readErr
	}
	defer func() { readNetworkPortStatus = original }()

	reader := newDefaultNetworkPortStatusReader()
	if _, err := reader.Read(context.Background()); !errors.Is(err, readErr) {
		t.Fatalf("expected port status error, got %v", err)
	}
}

func TestDefaultNetworkPortMembershipReaderFiltersDockerAndLoopback(t *testing.T) {
	networkPortListReaderTestMu.Lock()
	defer networkPortListReaderTestMu.Unlock()

	original := readNetworkPortMembershipDump
	readNetworkPortMembershipDump = func(ctx context.Context) ([]NetworkPortMembershipSnapshot, error) {
		return []NetworkPortMembershipSnapshot{
			{InterfaceName: "wan", Device: "eth0"},
			{InterfaceName: "docker", Device: "docker0"},
			{InterfaceName: "loopback", Device: "lo"},
			{InterfaceName: "lan", Device: "br-lan"},
		}, nil
	}
	defer func() { readNetworkPortMembershipDump = original }()

	reader := newDefaultNetworkPortMembershipReader()
	memberships, err := reader.Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected membership reader error: %v", err)
	}
	if len(memberships) != 2 {
		t.Fatalf("expected docker/loopback to be filtered, got %#v", memberships)
	}
	if memberships[0].InterfaceName != "wan" || memberships[1].InterfaceName != "lan" {
		t.Fatalf("unexpected filtered memberships: %#v", memberships)
	}
}

func TestDefaultNetworkPortMembershipReaderMapsLegacyError(t *testing.T) {
	networkPortListReaderTestMu.Lock()
	defer networkPortListReaderTestMu.Unlock()

	original := readNetworkPortMembershipDump
	readNetworkPortMembershipDump = func(ctx context.Context) ([]NetworkPortMembershipSnapshot, error) {
		return nil, errors.New("ubus failed")
	}
	defer func() { readNetworkPortMembershipDump = original }()

	reader := newDefaultNetworkPortMembershipReader()
	if _, err := reader.Read(context.Background()); err == nil || err.Error() != "获取网络接口失败" {
		t.Fatalf("expected legacy membership error, got %v", err)
	}
}

func TestReadNetworkPortMembershipDumpReadsUbusInterfaceDevices(t *testing.T) {
	networkPortListReaderTestMu.Lock()
	defer networkPortListReaderTestMu.Unlock()

	original := readNetworkPortMembershipUbusCall
	readNetworkPortMembershipUbusCall = func(ctx context.Context, arg string) (*simplejson.Json, error) {
		if arg != "network.interface dump" {
			t.Fatalf("unexpected ubus arg: %s", arg)
		}
		return simplejson.NewJson([]byte(`{
			"interface": [
				{"interface": "lan", "device": "br-lan"},
				{"interface": "wan", "device": "eth0"},
				{"interface": "wan6", "device": ""}
			]
		}`))
	}
	defer func() { readNetworkPortMembershipUbusCall = original }()

	memberships, err := readNetworkPortMembershipDump(context.Background())
	if err != nil {
		t.Fatalf("unexpected membership dump error: %v", err)
	}
	if len(memberships) != 3 {
		t.Fatalf("expected three memberships, got %#v", memberships)
	}
	if memberships[0] != (NetworkPortMembershipSnapshot{InterfaceName: "lan", Device: "br-lan"}) {
		t.Fatalf("unexpected first membership: %#v", memberships[0])
	}
	if memberships[1] != (NetworkPortMembershipSnapshot{InterfaceName: "wan", Device: "eth0"}) {
		t.Fatalf("unexpected second membership: %#v", memberships[1])
	}
	if memberships[2] != (NetworkPortMembershipSnapshot{InterfaceName: "wan6", Device: ""}) {
		t.Fatalf("unexpected third membership: %#v", memberships[2])
	}
}
