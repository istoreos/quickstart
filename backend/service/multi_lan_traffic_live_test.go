package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/istoreos/quickstart/backend/models"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// This entrypoint runs only inside the disposable router namespace created by
// multi-lan-traffic-probe.py. The real policy, quota, UCI and traffic adapters
// execute there; only inventory input and firewall reload location are supplied
// by the isolated harness. No global service is restarted.
func TestMultiLANLiveTrafficPolicy(t *testing.T) {
	base := os.Getenv("QUICKSTART_MULTI_LAN_TRAFFIC_LIVE_BASE")
	if base == "" {
		t.Skip("requires explicit isolated packet harness")
	}
	if !strings.HasPrefix(base, "/tmp/quickstart-multi-lan/m07-") {
		t.Fatal("unsafe live fixture directory")
	}
	marker, err := os.ReadFile(filepath.Join(base, "owner"))
	if err != nil || string(marker) != "M07 isolated netns" {
		t.Fatal("missing isolated fixture marker")
	}
	current, _ := os.Readlink("/proc/self/ns/net")
	initial, _ := os.Readlink("/proc/1/ns/net")
	if current == initial {
		t.Fatal("live firewall test requires isolated network namespace")
	}
	phase := os.Getenv("QUICKSTART_MULTI_LAN_TRAFFIC_LIVE_PHASE")
	if strings.HasPrefix(phase, "legacy-") {
		runLegacyQuotaPackets(t, base, phase)
		return
	}
	dir := filepath.Join(base, "config")
	network := os.Getenv("QUICKSTART_MULTI_LAN_TRAFFIC_LIVE_NETWORK")
	if network != "test_a" && network != "test_b" {
		t.Fatal("invalid isolated network")
	}
	device := "lana"
	ip := "192.0.2.55"
	if network == "test_b" {
		device = "lanb"
		ip = "198.51.100.55"
	}
	ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: network, Device: device, State: "up", FirewallZones: []string{"shared"}})
	const mac = "02:00:00:00:00:55"
	const id = "mac:02:00:00:00:00:55"
	provider := NewNativeRateLimitProvider("http://127.0.0.1:1", nil).(*nativeRateLimitProvider)
	provider.registry = &nativeEngineRegistry{configDir: dir}
	provider.offload = nil
	if phase == "rate" {
		rate, err := strconv.ParseInt(os.Getenv("QUICKSTART_MULTI_LAN_TRAFFIC_LIVE_RATE"), 10, 64)
		if err != nil || rate < 0 || rate > 100 {
			t.Fatal("invalid live rate")
		}
		if err := provider.Apply(ctx, rateLimitTarget{MAC: mac}, models.DeviceSpeedPolicy{Enabled: rate > 0, UploadSpeed: rate, DownloadSpeed: rate * 2}); err != nil {
			t.Fatal(err)
		}
		return
	}
	if phase == "fixtures" {
		for _, scope := range []*models.LANScope{{NetworkID: "test_a", Device: "lana", FirewallZones: []string{"shared"}}, {NetworkID: "test_b", Device: "lanb", FirewallZones: []string{"shared"}}} {
			if err := writeDeviceAccessPolicyAt(withLANScope(ctx, scope), dir, mac, true); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	now := time.Now()
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{ARP: []deviceInventoryObservation{{NetworkID: network, MAC: mac, IPv4: ip, Online: true}}, NetworkDevices: map[string]string{network: device}}}}
	inventory := newTestDeviceInventory(t, source, &now, 100)
	sampler := scopedDeviceTrafficSampler{native: provider}
	traffic := newDeviceTrafficModuleForTest(inventory, sampler, time.Now)
	snapshot, err := traffic.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if phase == "sample" {
		raw, _ := json.Marshal(snapshot)
		fmt.Printf("TRAFFIC=%s\n", raw)
		return
	}
	oldDir, oldApply := deviceRestrictionConfigDir, deviceAccessRestrictionApply
	deviceRestrictionConfigDir = func() string { return dir }
	deviceAccessRestrictionApply = func(ctx context.Context) error {
		out, err := exec.CommandContext(ctx, "sh", filepath.Join(base, "load-firewall")).CombinedOutput()
		if err != nil {
			return fmt.Errorf("isolated fw4 load: %w: %s", err, out)
		}
		return nil
	}
	defer func() { deviceRestrictionConfigDir, deviceAccessRestrictionApply = oldDir, oldApply }()
	policy := newDevicePolicyModuleForTest(&systemDevicePolicyStore{inventory: inventory})
	if phase == "block" || phase == "allow" {
		result, err := policy.Apply(ctx, &models.DevicePolicyApplyRequest{DeviceID: id, Kind: "access", Access: &models.DeviceAccessPolicy{NetworkAccess: phase == "allow"}})
		if err != nil || result.Result.Error != nil {
			t.Fatalf("access result=%+v err=%v", result, err)
		}
		return
	}
	quota := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(base, "traffic.json")), policy)
	quota.collector = traffic.Snapshot
	if phase == "quota-next-period" {
		quota.now = func() time.Time { return time.Now().Add(24 * time.Hour) }
		if err := quota.Record(ctx, snapshot); err != nil {
			t.Fatal(err)
		}
		status, err := quota.Get(ctx, id, "today")
		if err != nil || status.Result.Quota.Exceeded {
			t.Fatalf("quota period did not reset: %+v %v", status, err)
		}
	} else if phase == "quota-start" {
		if err := quota.Record(ctx, snapshot); err != nil {
			t.Fatal(err)
		}
		result, err := quota.SetQuota(ctx, &models.TrafficQuotaRequest{DeviceID: id, Enabled: true, Period: "daily", LimitBytes: 1024 * 1024, Action: "block"})
		if err != nil || result.Result.Error != nil {
			t.Fatalf("quota start result=%+v err=%v", result, err)
		}
	} else if phase == "quota-cancel" {
		result, err := quota.SetQuota(ctx, &models.TrafficQuotaRequest{DeviceID: id, Enabled: false})
		if err != nil || result.Result.Error != nil {
			t.Fatalf("quota cancel result=%+v err=%v", result, err)
		}
	} else if phase == "quota-record" {
		if err := quota.Record(ctx, snapshot); err != nil {
			t.Fatal(err)
		}
		status, err := quota.Get(ctx, id, "today")
		if err != nil || !status.Result.Quota.Exceeded {
			t.Fatalf("quota not exceeded: %+v %v", status, err)
		}
	} else {
		t.Fatal("unknown live phase", phase)
	}
}
