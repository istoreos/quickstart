package service

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

type fakeLanDHCPSettingsStore struct {
	snapshot lanDHCPSettingsSnapshot
	applies  int
	err      error
}

func (store *fakeLanDHCPSettingsStore) Read(context.Context) (lanDHCPSettingsSnapshot, error) {
	return store.snapshot, store.err
}

func (store *fakeLanDHCPSettingsStore) Apply(_ context.Context, plan lanDHCPSettingsExecutionPlan) error {
	if store.err != nil {
		return store.err
	}
	store.applies++
	store.snapshot.Settings = plan.Desired
	return nil
}

func testLanDHCPSettingsSnapshot() lanDHCPSettingsSnapshot {
	return lanDHCPSettingsSnapshot{
		Settings: &models.LanDHCPSettings{Enabled: true, PoolStart: "192.168.30.100", PoolEnd: "192.168.30.239", LeaseTime: "12h", DefaultTargetID: "self"},
		Prefix:   netip.MustParsePrefix("192.168.30.0/24"), RouterAddress: "192.168.30.1", Authority: "local", Version: "v1", AffectedDevices: 12,
		ProtectedAddresses: map[string]string{"192.168.30.3": "floating_gateway", "192.168.30.93": "address_reservation"},
		Targets:            map[string]string{"self": "192.168.30.1", "bypass:1": "192.168.30.244"},
	}
}

func TestLanDHCPSettingsPlanRejectsUnsafePoolWithoutWriting(t *testing.T) {
	store := &fakeLanDHCPSettingsStore{snapshot: testLanDHCPSettingsSnapshot()}
	module := NewLanDHCPSettingsModule(store)
	response, err := module.Plan(context.Background(), &models.LanDHCPSettingsApplyRequest{
		Settings: &models.LanDHCPSettings{Enabled: true, PoolStart: "192.168.30.2", PoolEnd: "192.168.30.120", LeaseTime: "12h", DefaultTargetID: "self"},
	})
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "address_conflict" || len(response.Result.Conflicts) != 2 || store.applies != 0 {
		t.Fatalf("plan=%#v applies=%d err=%v", response, store.applies, err)
	}
}

func TestLanDHCPSettingsDisableRequiresConfirmationAndReportsRecovery(t *testing.T) {
	store := &fakeLanDHCPSettingsStore{snapshot: testLanDHCPSettingsSnapshot()}
	module := NewLanDHCPSettingsModule(store)
	request := &models.LanDHCPSettingsApplyRequest{Settings: &models.LanDHCPSettings{Enabled: false, PoolStart: "192.168.30.100", PoolEnd: "192.168.30.239", LeaseTime: "12h", DefaultTargetID: "self"}}
	planned, err := module.Plan(context.Background(), request)
	if err != nil || planned.Result.Error == nil || planned.Result.Error.Code != "confirmation_required" || planned.Result.AffectedDevices != 12 || planned.Result.RecoveryGuidance == "" {
		t.Fatalf("plan=%#v err=%v", planned, err)
	}
	request.ConfirmDisable = true
	planned, err = module.Plan(context.Background(), request)
	if err != nil || planned.Result.Error != nil || !planned.Result.CanApply || store.applies != 0 {
		t.Fatalf("confirmed plan=%#v applies=%d err=%v", planned, store.applies, err)
	}
}

func TestLanDHCPSettingsApplyRejectsExternalAuthorityAndStaleVersion(t *testing.T) {
	state := testLanDHCPSettingsSnapshot()
	state.Authority = "external_observed"
	store := &fakeLanDHCPSettingsStore{snapshot: state}
	module := NewLanDHCPSettingsModule(store)
	request := &models.LanDHCPSettingsApplyRequest{ExpectedVersion: "stale", Settings: &models.LanDHCPSettings{Enabled: true, PoolStart: "192.168.30.120", PoolEnd: "192.168.30.220", LeaseTime: "12h", DefaultTargetID: "self"}}
	response, err := module.Apply(context.Background(), request)
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "dhcp_authority_unavailable" || store.applies != 0 {
		t.Fatalf("external apply=%#v applies=%d err=%v", response, store.applies, err)
	}
	store.snapshot.Authority = "local"
	response, err = module.Apply(context.Background(), request)
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "conflict" || store.applies != 0 {
		t.Fatalf("stale apply=%#v applies=%d err=%v", response, store.applies, err)
	}
}

func TestLanDHCPSettingsStoreReportsRollbackFailure(t *testing.T) {
	originalSnapshot, originalMutate, originalReload, originalRestore := lanDHCPSettingsTakeSnapshot, lanDHCPSettingsMutate, lanDHCPSettingsReload, lanDHCPSettingsRestore
	t.Cleanup(func() {
		lanDHCPSettingsTakeSnapshot, lanDHCPSettingsMutate, lanDHCPSettingsReload, lanDHCPSettingsRestore = originalSnapshot, originalMutate, originalReload, originalRestore
	})
	lanDHCPSettingsTakeSnapshot = func() (dhcpConfigSnapshot, error) { return dhcpConfigSnapshot{}, nil }
	lanDHCPSettingsMutate = func(lanDHCPSettingsExecutionPlan) error { return errors.New("write failed") }
	lanDHCPSettingsRestore = func(context.Context, dhcpConfigSnapshot) error { return errors.New("restore failed") }
	err := (&defaultLanDHCPSettingsStore{}).Apply(context.Background(), lanDHCPSettingsExecutionPlan{Desired: &models.LanDHCPSettings{}})
	if err == nil || !errors.Is(err, errLanDHCPRecoveryRequired) {
		t.Fatalf("rollback error=%v", err)
	}
}

func TestLanDHCPSettingsWriterCompilesPoolLeaseAndDNSFollowingRoute(t *testing.T) {
	directory := t.TempDir()
	config := "config dhcp 'lan'\n\toption interface 'lan'\n\toption start '10'\n\toption limit '20'\n\toption leasetime '12h'\n"
	if err := os.WriteFile(filepath.Join(directory, "dhcp"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	plan := lanDHCPSettingsExecutionPlan{
		State:          lanDHCPSettingsSnapshot{Prefix: netip.MustParsePrefix("192.168.30.0/24"), RouterAddress: "192.168.30.1"},
		Desired:        &models.LanDHCPSettings{Enabled: true, PoolStart: "192.168.30.100", PoolEnd: "192.168.30.150", LeaseTime: "6h", DefaultTargetID: "bypass:1"},
		DefaultGateway: "192.168.30.244",
	}
	if err := mutateLanDHCPSettingsAt(directory, plan); err != nil {
		t.Fatal(err)
	}
	tree := uci.NewTree(directory)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		t.Fatal(err)
	}
	start, _ := tree.GetLast("dhcp", "lan", "start")
	limit, _ := tree.GetLast("dhcp", "lan", "limit")
	lease, _ := tree.GetLast("dhcp", "lan", "leasetime")
	options, _ := tree.Get("dhcp", "lan", "dhcp_option")
	if start != "100" || limit != "51" || lease != "6h" || len(options) != 2 || options[0] != "3,192.168.30.244" || options[1] != "6,192.168.30.244" {
		t.Fatalf("start=%q limit=%q lease=%q options=%#v", start, limit, lease, options)
	}
}
