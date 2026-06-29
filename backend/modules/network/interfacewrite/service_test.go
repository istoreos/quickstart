package interfacewrite

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStore struct {
	snapshot       Snapshot
	readErr        error
	normalized     map[string][]string
	appliedPlan    CommandPlan
	applyErr       error
	normalizeCalls []string
}

func (store *fakeStore) ReadSnapshot(ctx context.Context) (Snapshot, error) {
	return store.snapshot, store.readErr
}

func (store *fakeStore) NormalizeDevices(name string, devices []string) []string {
	store.normalizeCalls = append(store.normalizeCalls, name)
	if store.normalized != nil {
		if got, ok := store.normalized[name]; ok {
			return append([]string(nil), got...)
		}
	}
	return append([]string(nil), devices...)
}

func (store *fakeStore) ApplyPlan(ctx context.Context, plan CommandPlan) error {
	store.appliedPlan = plan
	return store.applyErr
}

type fakeApply struct {
	configs []string
	err     error
}

func (apply *fakeApply) Apply(ctx context.Context, configs []string) error {
	apply.configs = append([]string(nil), configs...)
	return apply.err
}

func TestServiceBuildsDeleteBridgeAndFirewallPlans(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		snapshot: Snapshot{
			Interfaces: []*models.NetworkInterfaceInfo{
				{Name: "lan"},
				{Name: "wan"},
				{Name: "wan6"},
			},
			DeviceSections: []DeviceSnapshot{
				{SectionName: "cfg01", Name: "br-lan"},
			},
			FirewallZones: []FirewallZoneSnapshot{
				{SectionName: "cfg10", Name: "lan", Networks: []string{"lan"}},
				{SectionName: "cfg11", Name: "wan", Networks: []string{"wan"}},
			},
		},
		normalized: map[string][]string{
			"lan": {"eth0", "dsm-ext"},
		},
	}
	apply := &fakeApply{}
	svc := NewService(store, apply)

	resp, err := svc.ApplyConfigSet(context.Background(), Input{
		Configs: []*models.NetworkInterfaceConfig{
			{Name: "lan", Proto: "dhcp", Devices: []string{"eth0"}, FirewallType: "lan"},
			{Name: "wan", Proto: "pppoe", Devices: []string{"eth1"}, FirewallType: "wan"},
			{Name: "guest", Proto: "dhcp", Devices: []string{"eth2", "eth3"}, FirewallType: "lan"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if resp == nil || resp.Success == nil {
		t.Fatalf("expected normal success response, got %#v", resp)
	}
	if !reflect.DeepEqual(store.normalizeCalls, []string{"lan", "wan", "guest"}) {
		t.Fatalf("unexpected normalize calls: %#v", store.normalizeCalls)
	}
	if !reflect.DeepEqual(store.appliedPlan.DeleteCommands, []string{"uci del network.wan6"}) {
		t.Fatalf("unexpected delete commands: %#v", store.appliedPlan.DeleteCommands)
	}
	if len(store.appliedPlan.BridgeCommands) != 2 {
		t.Fatalf("expected 2 bridge command batches, got %#v", store.appliedPlan.BridgeCommands)
	}
	if !reflect.DeepEqual(store.appliedPlan.BridgeCommands[0], []string{
		"uci del network.cfg01.ports",
		"uci del network.cfg01.ports",
		"uci add_list network.cfg01.ports=eth0",
		"uci add_list network.cfg01.ports=dsm-ext",
	}) {
		t.Fatalf("unexpected lan bridge plan: %#v", store.appliedPlan.BridgeCommands[0])
	}
	if !reflect.DeepEqual(store.appliedPlan.FirewallCommands, [][]string{
		{"uci add_list firewall.cfg10.network=guest"},
	}) {
		t.Fatalf("unexpected firewall commands: %#v", store.appliedPlan.FirewallCommands)
	}
	if !reflect.DeepEqual(apply.configs, []string{"firewall", "network"}) {
		t.Fatalf("unexpected apply configs: %#v", apply.configs)
	}
}

func TestServicePropagatesStoreErrors(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		snapshot: Snapshot{},
		applyErr: errors.New("store failed"),
	}
	svc := NewService(store, &fakeApply{})

	_, err := svc.ApplyConfigSet(context.Background(), Input{
		Configs: []*models.NetworkInterfaceConfig{{Name: "lan", Proto: "dhcp", Devices: []string{"eth0"}, FirewallType: "lan"}},
	})
	if !errors.Is(err, store.applyErr) {
		t.Fatalf("expected store error, got %v", err)
	}
}

func TestServicePropagatesApplyErrors(t *testing.T) {
	t.Parallel()

	applyErr := errors.New("apply failed")
	svc := NewService(&fakeStore{snapshot: Snapshot{}}, &fakeApply{err: applyErr})

	_, err := svc.ApplyConfigSet(context.Background(), Input{
		Configs: []*models.NetworkInterfaceConfig{{Name: "wan", Proto: "dhcp", Devices: []string{"eth0"}, FirewallType: "wan"}},
	})
	if !errors.Is(err, applyErr) {
		t.Fatalf("expected apply error, got %v", err)
	}
}

func TestServicePropagatesReadErrors(t *testing.T) {
	t.Parallel()

	readErr := errors.New("read failed")
	svc := NewService(&fakeStore{readErr: readErr}, &fakeApply{})

	_, err := svc.ApplyConfigSet(context.Background(), Input{})
	if !errors.Is(err, readErr) {
		t.Fatalf("expected read error, got %v", err)
	}
}
