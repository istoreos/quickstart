package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

type fakeFloatingGatewayStore struct {
	state   floatingGatewaySnapshot
	applies int
	err     error
}

func (store *fakeFloatingGatewayStore) Read(context.Context) (floatingGatewaySnapshot, error) {
	return store.state, store.err
}
func (store *fakeFloatingGatewayStore) Apply(_ context.Context, config *models.FloatingGatewayConfig) error {
	if store.err != nil {
		return store.err
	}
	store.applies++
	store.state.Config = normalizedFloatingConfig(config)
	store.state.Running = config.Enabled
	store.state.LocalHolder = config.Enabled
	store.state.Version = "version-two"
	return nil
}

func floatingGatewayTestState() floatingGatewaySnapshot {
	return floatingGatewaySnapshot{Installed: true, Config: &models.FloatingGatewayConfig{PeerIPs: []string{}, HealthTimeoutSec: 5}, Prefixes: []netip.Prefix{netip.MustParsePrefix("192.168.100.0/24")}, UsedIPs: map[string]bool{"192.168.100.20": true}, Version: "version-one"}
}

func validFloatingGatewayConfig() *models.FloatingGatewayConfig {
	return &models.FloatingGatewayConfig{Enabled: true, Role: "preferred", VirtualIP: "192.168.100.99", PeerIPs: []string{"192.168.100.2"}, HealthURL: "https://example.com/health", HealthTimeoutSec: 5}
}

func TestFloatingGatewayUsesProductRolesAndCapabilityState(t *testing.T) {
	state := floatingGatewayTestState()
	state.Config = validFloatingGatewayConfig()
	state.Running = true
	state.LocalHolder = true
	response, err := NewFloatingGatewayModule(&fakeFloatingGatewayStore{state: state}, nil).Get(context.Background())
	if err != nil || response.Result.Status.State != "healthy" || response.Result.Status.Holder != "local" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	raw, _ := json.Marshal(response)
	if strings.Contains(string(raw), `"main"`) || strings.Contains(string(raw), `"fallback"`) {
		t.Fatalf("internal role leaked: %s", raw)
	}
	state.Installed = false
	missing, _ := NewFloatingGatewayModule(&fakeFloatingGatewayStore{state: state}, nil).Get(context.Background())
	if missing.Result.Status.Capability != "not_installed" || missing.Result.Status.Reason != "dependency_not_installed" {
		t.Fatalf("missing=%#v", missing.Result.Status)
	}
}

func TestFloatingGatewayPlanValidatesBeforeWrite(t *testing.T) {
	store := &fakeFloatingGatewayStore{state: floatingGatewayTestState()}
	module := NewFloatingGatewayModule(store, nil)
	tests := []struct {
		name   string
		mutate func(*models.FloatingGatewayConfig)
	}{
		{"outside LAN", func(config *models.FloatingGatewayConfig) { config.VirtualIP = "203.0.113.9" }},
		{"address occupied", func(config *models.FloatingGatewayConfig) { config.VirtualIP = "192.168.100.20" }},
		{"invalid URL", func(config *models.FloatingGatewayConfig) { config.HealthURL = "file:///etc/passwd" }},
		{"unsafe URL", func(config *models.FloatingGatewayConfig) { config.HealthURL = "https://example.com/'bad" }},
		{"takeover requires peer", func(config *models.FloatingGatewayConfig) { config.Role = "takeover"; config.PeerIPs = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validFloatingGatewayConfig()
			tt.mutate(config)
			response, err := module.Plan(context.Background(), &models.FloatingGatewayApplyRequest{Config: config})
			if err != nil || response.Result.Plan.Error == nil || store.applies != 0 {
				t.Fatalf("response=%#v applies=%d err=%v", response, store.applies, err)
			}
		})
	}
}

func TestFloatingGatewayNormalizesLegacyCIDRAndBlocksDisabledAddressCollision(t *testing.T) {
	state := floatingGatewayTestState()
	state.Config = &models.FloatingGatewayConfig{
		Enabled:          false,
		Role:             "takeover",
		VirtualIP:        "192.168.100.20/24",
		PeerIPs:          []string{"192.168.100.2"},
		HealthTimeoutSec: 5,
	}
	state.Config = normalizedFloatingConfig(state.Config)
	if state.Config.VirtualIP != "192.168.100.20" {
		t.Fatalf("normalized virtual IP=%q", state.Config.VirtualIP)
	}

	response, err := NewFloatingGatewayModule(&fakeFloatingGatewayStore{state: state}, nil).Plan(context.Background(), &models.FloatingGatewayApplyRequest{Config: &models.FloatingGatewayConfig{
		Enabled:          true,
		Role:             "takeover",
		VirtualIP:        "192.168.100.20",
		PeerIPs:          []string{"192.168.100.2"},
		HealthTimeoutSec: 5,
	}})
	if err != nil || response.Result.Plan.Error == nil || response.Result.Plan.Error.Code != "conflict" || response.Result.Plan.CanApply {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}

func TestFloatingGatewayApplyChecksVersionAndIsIdempotent(t *testing.T) {
	store := &fakeFloatingGatewayStore{state: floatingGatewayTestState()}
	module := NewFloatingGatewayModule(store, nil)
	stale, _ := module.Apply(context.Background(), &models.FloatingGatewayApplyRequest{Config: validFloatingGatewayConfig(), ExpectedVersion: "stale"})
	if stale.Result.Plan.Error == nil || stale.Result.Plan.Error.Code != "conflict" || store.applies != 0 {
		t.Fatalf("stale=%#v applies=%d", stale.Result, store.applies)
	}
	applied, _ := module.Apply(context.Background(), &models.FloatingGatewayApplyRequest{Config: validFloatingGatewayConfig(), ExpectedVersion: "version-one"})
	if !applied.Result.Changed || applied.Result.Status.State != "healthy" || store.applies != 1 {
		t.Fatalf("applied=%#v applies=%d", applied.Result, store.applies)
	}
	repeated, _ := module.Apply(context.Background(), &models.FloatingGatewayApplyRequest{Config: validFloatingGatewayConfig(), ExpectedVersion: "version-two"})
	if repeated.Result.Changed || store.applies != 1 {
		t.Fatalf("repeated=%#v applies=%d", repeated.Result, store.applies)
	}
}

func TestFloatingGatewayBlocksReferencedDisable(t *testing.T) {
	gatewayState := gatewayPolicyTestState()
	gatewayState.DHCP.FloatIP = &FloatIPSnapshot{Enabled: true, SetIP: "192.168.100.99", CheckIP: "192.168.100.2"}
	gatewayState.Hosts[0].TagName = ipToDhcpTag("192.168.100.99")
	gateway := gatewayPolicyTestModule(t, &fakeGatewayPolicyStore{state: gatewayState})
	state := floatingGatewayTestState()
	state.Config = validFloatingGatewayConfig()
	response, err := NewFloatingGatewayModule(&fakeFloatingGatewayStore{state: state}, gateway).Plan(context.Background(), &models.FloatingGatewayApplyRequest{Config: &models.FloatingGatewayConfig{Enabled: false}})
	if err != nil || response.Result.Plan.Error == nil || response.Result.Plan.Error.Code != "conflict" || len(response.Result.Plan.AffectedDevices) != 1 {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}

func TestMutateFloatingGatewayConfigUsesTypedUCIAndPreservesUnknownOption(t *testing.T) {
	directory := t.TempDir()
	config := "config floatip 'main'\n\toption enabled '0'\n\toption vendor_extension 'keep'\n"
	if err := os.WriteFile(filepath.Join(directory, "floatip"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	if err := mutateFloatingGatewayConfigAt(directory, validFloatingGatewayConfig()); err != nil {
		t.Fatal(err)
	}
	tree := uci.NewTree(directory)
	if err := tree.LoadConfig("floatip", true); err != nil {
		t.Fatal(err)
	}
	for option, want := range map[string]string{"enabled": "1", "role": "main", "set_ip": "192.168.100.99", "vendor_extension": "keep"} {
		if got, _ := tree.GetLast("floatip", "main", option); got != want {
			t.Fatalf("%s=%q want %q", option, got, want)
		}
	}
	peers, _ := tree.Get("floatip", "main", "check_ip")
	if len(peers) != 1 || peers[0] != "192.168.100.2" {
		t.Fatalf("peers=%#v", peers)
	}
}

func TestFloatingGatewayStoreRollsBackReloadFailure(t *testing.T) {
	originalSnapshot, originalMutate, originalReload, originalRestore := floatingGatewaySnapshotConfig, floatingGatewayMutateConfig, floatingGatewayReload, floatingGatewayRestoreConfig
	t.Cleanup(func() {
		floatingGatewaySnapshotConfig, floatingGatewayMutateConfig, floatingGatewayReload, floatingGatewayRestoreConfig = originalSnapshot, originalMutate, originalReload, originalRestore
	})
	floatingGatewaySnapshotConfig = func() (floatingGatewayConfigSnapshot, error) {
		return floatingGatewayConfigSnapshot{Data: []byte("before"), Mode: 0600, Exists: true}, nil
	}
	floatingGatewayMutateConfig = func(*models.FloatingGatewayConfig) error { return nil }
	floatingGatewayReload = func(context.Context, bool) error { return errors.New("service failed") }
	restored := false
	floatingGatewayRestoreConfig = func(context.Context, floatingGatewayConfigSnapshot) error { restored = true; return nil }
	err := (&systemFloatingGatewayStore{}).Apply(context.Background(), validFloatingGatewayConfig())
	if err == nil || !restored {
		t.Fatalf("err=%v restored=%v", err, restored)
	}
}
