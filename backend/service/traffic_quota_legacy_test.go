package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

func TestLegacyQuotaRequiresExplicitAccessRecovery(t *testing.T) {
	for _, action := range []string{"cancel", "period_reset", "current_period_exceeded", "change_to_notify", "missing_runtime", "missing_runtime_offline", "missing_runtime_period_reset", "missing_runtime_notify", "missing_runtime_group", "mixed_owned_rule"} {
		t.Run(action, func(t *testing.T) {
			dir := t.TempDir()
			oldDir, oldApply, oldRestriction := deviceRestrictionConfigDir, deviceAccessRestrictionApply, deviceRestrictionApply
			deviceRestrictionConfigDir = func() string { return dir }
			reloads := 0
			deviceAccessRestrictionApply = func(context.Context) error { reloads++; return nil }
			deviceRestrictionApply = func(context.Context, []string) error { return nil }
			defer func() {
				deviceRestrictionConfigDir, deviceAccessRestrictionApply, deviceRestrictionApply = oldDir, oldApply, oldRestriction
			}()
			const mac = "02:00:00:00:00:55"
			const id = "mac:02:00:00:00:00:55"
			raw := fmt.Sprintf("config zone 'local'\n option name 'lan'\nconfig zone 'outside'\n option name 'wan'\nconfig rule 'old'\n option name '%s'\n option src 'lan'\n option dest 'wan'\n option src_mac '%s'\n option target 'REJECT'\n", buildBlockedRuleName(mac), mac)
			if err := os.WriteFile(filepath.Join(dir, "firewall"), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
			ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: "lan", Device: "br-lan", State: "up", FirewallZones: []string{"lan"}})
			office := withLANScope(context.Background(), &models.LANScope{NetworkID: "office", Device: "br-office", State: "up", FirewallZones: []string{"lan"}})
			inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{ARP: []deviceInventoryObservation{{NetworkID: "lan", MAC: mac, IPv4: "192.0.2.55", Online: true}, {NetworkID: "office", MAC: mac, IPv4: "198.51.100.55", Online: true}}}}}, &now, 100)
			policy := newDevicePolicyModuleForTest(&systemDevicePolicyStore{inventory: inventory})
			store := newTrafficInsightsFileStore(filepath.Join(dir, "traffic.json"))
			module := NewTrafficInsightsModule(store, policy)
			module.now = func() time.Time { return now }
			if err := module.load(); err != nil {
				t.Fatal(err)
			}
			start, _ := trafficQuotaPeriod(now, "daily")
			module.document.Quotas[id] = &models.TrafficQuota{DeviceID: id, Enabled: true, Period: "daily", LimitBytes: 100, Action: "block"}
			module.document.QuotaRuntime[id] = trafficQuotaRuntime{PeriodStart: start.Format(time.RFC3339), Blocked: true, RestoreAccess: true}
			if action == "current_period_exceeded" {
				module.document.Daily[id] = []*models.TrafficInsightBucket{{Start: start.Format(time.RFC3339), UploadBytes: 200}}
			}
			if strings.HasPrefix(action, "missing_runtime") {
				delete(module.document.QuotaRuntime, id)
			}
			if action == "missing_runtime_offline" {
				policy.store.(*systemDevicePolicyStore).inventory = nil
			}
			if action == "mixed_owned_rule" {
				if err := writeDeviceAccessPolicyAt(context.WithValue(ctx, accessOwnerKey{}, "quota"), dir, mac, false); err != nil {
					t.Fatal(err)
				}
			}
			if err := module.persist(now); err != nil {
				t.Fatal(err)
			}
			switch action {
			case "current_period_exceeded":
				for i := 0; i < 2; i++ {
					if err := module.Record(ctx, &models.DeviceTrafficResponse{Result: &models.DeviceTrafficResult{}}); err != nil {
						t.Fatal(err)
					}
				}
			case "cancel", "missing_runtime", "missing_runtime_offline", "mixed_owned_rule":
				result, err := module.SetQuota(ctx, &models.TrafficQuotaRequest{DeviceID: id, Enabled: false})
				if err != nil || result.Result.Error != nil || !result.Result.LegacyAccessReviewRequired {
					t.Fatal("cancel lost review", err, result)
				}
			case "period_reset", "missing_runtime_period_reset":
				now = now.Add(24 * time.Hour)
				if err := module.Record(ctx, &models.DeviceTrafficResponse{Result: &models.DeviceTrafficResult{}}); err != nil {
					t.Fatal(err)
				}
			case "change_to_notify", "missing_runtime_notify":
				result, err := module.SetQuota(ctx, &models.TrafficQuotaRequest{DeviceID: id, Enabled: true, Period: "daily", LimitBytes: 1000, Action: "notify"})
				if err != nil || result.Result.Error != nil {
					t.Fatal(err, result)
				}
			case "missing_runtime_group":
				applier := &defaultGroupPolicyApplier{policy: policy, traffic: module}
				result := applier.ApplyBatch(ctx, []*models.EffectiveGroupPolicy{{DeviceID: id, Managed: []string{"quota"}, Quota: &models.GroupQuotaPolicy{Enabled: false}}}, now)
				if result.Failed != 0 || result.Reloads["firewall"] != 1 {
					t.Fatalf("quota-only group did not commit access binding: %+v", result)
				}
			}
			if reloads != 1 {
				t.Fatalf("expected one physical reload, got %d", reloads)
			}
			tree := uci.NewTree(dir)
			if err := tree.LoadConfig("firewall", true); err != nil {
				t.Fatal(err)
			}
			for key, expected := range map[string]string{"quickstart_network": "lan", "device": "br-lan", "direction": "in", "quickstart_owner": "", "quickstart_access_version": ""} {
				actual, _ := tree.GetLast("firewall", "old", key)
				if actual != expected {
					t.Fatalf("legacy binding %s = %q, expected %q", key, actual, expected)
				}
			}
			include, err := os.ReadFile(immediateAccessIncludePath(dir))
			if err != nil || !strings.Contains(string(include), `iifname "br-lan"`) {
				t.Fatalf("physical rule omitted original LAN: %s: %v", include, err)
			}
			allowed, err := readDeviceAccessPolicyAt(ctx, dir, mac)
			if err != nil || allowed {
				t.Fatal("legacy pause was released", err)
			}
			restarted := NewTrafficInsightsModule(newTrafficInsightsFileStore(store.path), policy)
			restarted.now = module.now
			result, err := restarted.Get(ctx, id, "today")
			if err != nil || !result.Result.LegacyAccessReviewRequired {
				t.Fatal("review lost on restart", err, result)
			}
			other, err := restarted.Get(office, id, "today")
			if err != nil || other.Result.LegacyAccessReviewRequired {
				t.Fatal("review leaked to another LAN", err, other)
			}
			// Background plans cannot quietly release or relabel the old rule.
			applier := &defaultGroupPolicyApplier{policy: policy, traffic: restarted}
			for _, access := range []bool{true, false} {
				if err := applier.Apply(ctx, &models.EffectiveGroupPolicy{DeviceID: id, NetworkAccess: access, Managed: []string{"access"}}); err != nil {
					t.Fatal(err)
				}
				result, err = restarted.Get(ctx, id, "today")
				if err != nil || !result.Result.LegacyAccessReviewRequired {
					t.Fatal("background plan replaced legacy pause", err, result)
				}
			}
			// This is the existing explicit access action, not a quota side effect.
			policy.store.(*systemDevicePolicyStore).inventory = inventory
			resultPolicy, err := policy.Apply(ctx, &models.DevicePolicyApplyRequest{DeviceID: id, Kind: "access", Access: &models.DeviceAccessPolicy{NetworkAccess: true}})
			if err != nil || resultPolicy.Result.Error != nil {
				t.Fatal(err, resultPolicy)
			}
			result, err = restarted.Get(ctx, id, "today")
			if err != nil || result.Result.LegacyAccessReviewRequired {
				t.Fatal("review survived explicit recovery", err, result)
			}
			if err := writeDeviceAccessPolicyAt(ctx, dir, mac, false); err != nil {
				t.Fatal(err)
			}
			result, err = restarted.Get(ctx, id, "today")
			if err != nil || result.Result.LegacyAccessReviewRequired {
				t.Fatal("new manual pause resurrected old warning", err, result)
			}
		})
	}
}

func TestLegacyQuotaBindingReloadFailureRestoresRuleAndQuota(t *testing.T) {
	for _, action := range []string{"cancel", "period_reset", "group"} {
		t.Run(action, func(t *testing.T) {
			dir := t.TempDir()
			oldDir, oldApply, oldRestriction := deviceRestrictionConfigDir, deviceAccessRestrictionApply, deviceRestrictionApply
			deviceRestrictionConfigDir = func() string { return dir }
			deviceRestrictionApply = func(context.Context, []string) error { return nil }
			reloads := 0
			deviceAccessRestrictionApply = func(context.Context) error {
				reloads++
				if reloads == 1 {
					return errors.New("injected firewall reload failure")
				}
				return nil
			}
			defer func() {
				deviceRestrictionConfigDir, deviceAccessRestrictionApply, deviceRestrictionApply = oldDir, oldApply, oldRestriction
			}()
			const mac = "02:00:00:00:00:55"
			const id = "mac:" + mac
			raw := fmt.Sprintf("config zone 'local'\n option name 'lan'\nconfig zone 'outside'\n option name 'wan'\nconfig rule 'old'\n option name '%s'\n option src 'lan'\n option dest 'wan'\n option src_mac '%s'\n option target 'REJECT'\n", buildBlockedRuleName(mac), mac)
			if err := os.WriteFile(filepath.Join(dir, "firewall"), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			now := time.Now()
			inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{ARP: []deviceInventoryObservation{{NetworkID: "lan", MAC: mac, IPv4: "192.0.2.55", Online: true}}}}}, &now, 100)
			policy := newDevicePolicyModuleForTest(&systemDevicePolicyStore{inventory: inventory})
			module := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(dir, "traffic.json")), policy)
			if err := module.load(); err != nil {
				t.Fatal(err)
			}
			module.document.Quotas[id] = &models.TrafficQuota{DeviceID: id, Enabled: true, Period: "daily", LimitBytes: 100, Action: "block"}
			if err := module.persist(now); err != nil {
				t.Fatal(err)
			}
			ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: "lan", Device: "br-lan", State: "up", FirewallZones: []string{"lan"}})
			switch action {
			case "cancel":
				if _, err := module.SetQuotaBatch(ctx, []*models.TrafficQuotaRequest{{DeviceID: id}}); err == nil {
					t.Fatal("reload failure was accepted")
				}
			case "period_reset":
				if err := module.Record(ctx, &models.DeviceTrafficResponse{Result: &models.DeviceTrafficResult{}}); err == nil {
					t.Fatal("reload failure was accepted")
				}
			case "group":
				applier := &defaultGroupPolicyApplier{policy: policy, traffic: module}
				result := applier.ApplyBatch(ctx, []*models.EffectiveGroupPolicy{{DeviceID: id, Managed: []string{"quota"}, Quota: &models.GroupQuotaPolicy{}}}, now)
				if result.Failed != 1 || result.Applied != 0 {
					t.Fatalf("reload failure was accepted: %+v", result)
				}
			}
			actual, err := os.ReadFile(filepath.Join(dir, "firewall"))
			if err != nil || string(actual) != raw || reloads != 2 {
				t.Fatalf("rule snapshot not restored: %s; reloads=%d; err=%v", actual, reloads, err)
			}
			document, _, err := module.store.Load()
			if err != nil || document.Quotas[id] == nil || !document.Quotas[id].Enabled || len(document.QuotaRuntime) != 0 {
				t.Fatal("quota snapshot not restored", err, document)
			}
		})
	}
}
