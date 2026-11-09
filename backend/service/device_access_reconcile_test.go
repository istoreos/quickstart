package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

func TestAccessBindingLifecycleStartupRebindAndReadOnlyObservation(t *testing.T) {
	dir := t.TempDir()
	raw := "config zone 'local'\n option name 'shared'\nconfig zone 'external'\n option name 'wan'\nconfig rule 'old'\n option name 'BL_old'\n option src 'shared'\n option dest 'wan'\n option src_mac '02:00:00:00:00:55'\n option target 'REJECT'\n"
	if err := os.WriteFile(filepath.Join(dir, "firewall"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	oldDir, oldApply := deviceRestrictionConfigDir, deviceAccessRestrictionApply
	deviceRestrictionConfigDir = func() string { return dir }
	reloads := 0
	deviceAccessRestrictionApply = func(context.Context) error { reloads++; return nil }
	defer func() { deviceRestrictionConfigDir, deviceAccessRestrictionApply = oldDir, oldApply }()
	lan := &models.LANScope{NetworkID: "lan", Device: "br-lan", State: "up", Version: "one", FirewallZones: []string{"shared"}}
	office := &models.LANScope{NetworkID: "office", Device: "br-office", State: "up", FirewallZones: []string{"shared"}}
	resolver := &LANScopeModule{reader: fakeLANScopeReader{scopes: []*models.LANScope{lan, office}}, now: time.Now}
	for _, scope := range []*models.LANScope{lan, office} {
		_, err := readDeviceAccessPolicyAt(withLANScope(context.Background(), scope), dir, "02:00:00:00:00:55")
		var pending *PolicyError
		if !errors.As(err, &pending) || pending.Code != "access_binding_pending" {
			t.Fatal("unbound rule falsely reported current", err)
		}
	}
	unchanged, _ := os.ReadFile(filepath.Join(dir, "firewall"))
	if string(unchanged) != raw || reloads != 0 {
		t.Fatal("read mutated network configuration")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	backend := &ServiceBackend{lanScopes: resolver}
	backend.StartLANPolicyReconciliation(ctx)
	bound, _ := os.ReadFile(filepath.Join(dir, "firewall"))
	if reloads != 1 || !strings.Contains(string(bound), "br-lan") {
		t.Fatal("startup did not narrow old manual pause", reloads, string(bound))
	}
	workerCtx := context.WithValue(context.Background(), lanScopeResolverKey{}, resolver)
	if err := reconcileDeviceAccessBindings(workerCtx); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(filepath.Join(dir, "firewall"))
	if reloads != 1 || string(again) != string(bound) {
		t.Fatal("unchanged binding rewritten/reloaded")
	}
	lan.Device, lan.Version = "br-new", "two"
	allowed, err := readDeviceAccessPolicyAt(withLANScope(context.Background(), lan), dir, "02:00:00:00:00:55")
	if allowed || err == nil {
		t.Fatal("stale desired pause falsely reported effective", allowed, err)
	}
	if err := reconcileDeviceAccessBindings(workerCtx); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		scope   *models.LANScope
		allowed bool
	}{{lan, false}, {office, true}} {
		allowed, err := readDeviceAccessPolicyAt(withLANScope(context.Background(), test.scope), dir, "02:00:00:00:00:55")
		if err != nil || allowed != test.allowed {
			t.Fatal("rebind changed another LAN", allowed, err)
		}
	}
	if reloads != 2 {
		t.Fatal("rebind did not use one reload", reloads)
	}
	legacy, err := legacyDeviceAccessRule(withLANScope(context.Background(), lan), dir, "02:00:00:00:00:55")
	if err != nil || !legacy {
		t.Fatal("reconciliation replaced legacy ownership/generation", err)
	}
}

func TestAccessBindingReconciliationRollback(t *testing.T) {
	dir := t.TempDir()
	raw := "config zone 'local'\n option name 'shared'\nconfig zone 'external'\n option name 'wan'\nconfig rule 'old'\n option name 'BL_old'\n option src 'shared'\n option dest 'wan'\n option src_mac '02:00:00:00:00:55'\n option target 'REJECT'\n"
	if err := os.WriteFile(filepath.Join(dir, "firewall"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	oldDir, oldApply := deviceRestrictionConfigDir, deviceAccessRestrictionApply
	deviceRestrictionConfigDir = func() string { return dir }
	reloads := 0
	deviceAccessRestrictionApply = func(context.Context) error {
		reloads++
		if reloads == 1 {
			return errors.New("injected reload failure")
		}
		return nil
	}
	defer func() { deviceRestrictionConfigDir, deviceAccessRestrictionApply = oldDir, oldApply }()
	ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: "lan", Device: "br-lan", State: "up", FirewallZones: []string{"shared"}})
	if err := reconcileDeviceAccessBindings(ctx); err == nil {
		t.Fatal("failed reload reported success")
	}
	actual, _ := os.ReadFile(filepath.Join(dir, "firewall"))
	if string(actual) != raw || reloads != 2 {
		t.Fatal("failed startup migration did not restore old rule", string(actual), reloads)
	}
}
