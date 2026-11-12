package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/istoreos/quickstart/backend/models"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func auxiliaryLiveBase(t *testing.T) string {
	t.Helper()
	base := os.Getenv("QUICKSTART_MULTI_LAN_AUX_LIVE_BASE")
	if base == "" {
		t.Skip("requires explicit M08 isolated packet harness")
	}
	marker, err := os.ReadFile(filepath.Join(base, "owner"))
	if err != nil || string(marker) != "M08 isolated netns" || !strings.HasPrefix(base, "/tmp/quickstart-multi-lan/m08-") {
		t.Fatal("invalid M08 fixture")
	}
	current, _ := os.Readlink("/proc/self/ns/net")
	initial, _ := os.Readlink("/proc/1/ns/net")
	if current == initial {
		t.Fatal("M08 packet entrypoint requires a private network namespace")
	}
	return base
}

func TestMultiLANLiveAuxiliaryHTTPServer(t *testing.T) {
	base := auxiliaryLiveBase(t)
	side := os.Getenv("QUICKSTART_MULTI_LAN_AUX_LIVE_SIDE")
	if side != "a" && side != "b" {
		t.Fatal("invalid server side")
	}
	var mu sync.Mutex
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		file, err := os.OpenFile(filepath.Join(base, "http-"+side+".jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			t.Error(err)
			return
		}
		defer file.Close()
		_ = json.NewEncoder(file).Encode(map[string]string{"side": side, "remote": r.RemoteAddr, "method": r.Method})
		w.WriteHeader(http.StatusNoContent)
	})
	for _, family := range []string{"tcp4", "tcp6"} {
		address := "0.0.0.0:8080"
		if family == "tcp6" {
			address = "[::]:8080"
		}
		listener, err := net.Listen(family, address)
		if err != nil {
			t.Fatal(err)
		}
		go http.Serve(listener, handler)
	}
	if err := os.WriteFile(filepath.Join(base, "http-"+side+".ready"), []byte("ready"), 0600); err != nil {
		t.Fatal(err)
	}
	select {}
}

func auxiliaryLiveInventory(t *testing.T) (*DeviceInventoryModule, []*models.LANScope) {
	t.Helper()
	now := time.Now()
	const mac = "02:00:00:00:00:55"
	snapshot := deviceInventorySourceSnapshot{NetworkDevices: map[string]string{"test_a": "lana", "test_b": "lanb"}}
	scopes := []*models.LANScope{}
	for _, side := range []string{"a", "b"} {
		device, prefix, v6 := "lana", "192.0.2", "fd71"
		if side == "b" {
			device, prefix, v6 = "lanb", "198.51.100", "fd72"
		}
		network := "test_" + side
		snapshot.ARP = append(snapshot.ARP, deviceInventoryObservation{NetworkID: network, MAC: mac, IPv4: prefix + ".55", Online: true})
		snapshot.NDP = append(snapshot.NDP, deviceInventoryObservation{NetworkID: network, MAC: mac, IPv6: v6 + "::55", Online: true})
		if link := os.Getenv("QUICKSTART_MULTI_LAN_AUX_LIVE_LINK_" + strings.ToUpper(side)); link != "" {
			snapshot.NDP = append(snapshot.NDP, deviceInventoryObservation{NetworkID: network, MAC: mac, IPv6: link, Online: true})
		}
		scopes = append(scopes, &models.LANScope{NetworkID: network, Device: device, State: "up", Version: "binding-" + side, IPv4Prefixes: []string{prefix + ".1/24"}, IPv6Prefixes: []string{v6 + "::1/64", "fe80::/64"}, FirewallZones: []string{"shared"}})
	}
	return newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, &now, 100), scopes
}

func TestMultiLANLiveAuxiliaryProbe(t *testing.T) {
	base := auxiliaryLiveBase(t)
	inventory, scopes := auxiliaryLiveInventory(t)
	probe := NewManagementProbeModule(inventory, NewNetworkAuditModule(filepath.Join(base, "audit.json"), nil))
	const id = "mac:02:00:00:00:00:55"
	for index, scope := range scopes {
		ctx := withLANScope(context.Background(), scope)
		addresses := []string{"192.0.2.55", "fd71::55"}
		other := "198.51.100.55"
		side := "A"
		if index == 1 {
			addresses = []string{"198.51.100.55", "fd72::55"}
			other = "192.0.2.55"
			side = "B"
		}
		if link := os.Getenv("QUICKSTART_MULTI_LAN_AUX_LIVE_LINK_" + side); link != "" {
			addresses = append(addresses, link)
		}
		for _, address := range addresses {
			response, err := probe.Probe(ctx, &models.ManagementProbeRequest{DeviceID: id, Address: address, Scheme: "http", Port: 8080})
			if err != nil || response.Result.Error != nil || !response.Result.Reachable {
				t.Fatalf("probe %s/%s: %+v %v", scope.NetworkID, address, response, err)
			}
		}
		rejected, err := probe.Probe(ctx, &models.ManagementProbeRequest{DeviceID: id, Address: other, Scheme: "http", Port: 8080})
		if err != nil || rejected.Result.Error == nil || rejected.Result.Error.Code != "validation_failed" {
			t.Fatal("probe escaped selected network", err)
		}
	}
}

func TestMultiLANLiveAuxiliaryConnection(t *testing.T) {
	base := auxiliaryLiveBase(t)
	_, scopes := auxiliaryLiveInventory(t)
	service := filepath.Join(base, "scoped-init")
	if err := os.WriteFile(service, []byte("# "+nativeScopeContract+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(base, "quickstart-netpolicy"))
	if err != nil {
		t.Fatal(err)
	}
	store := &systemCapabilityActionStore{configDir: filepath.Join(base, "config"), servicePath: service, stat: func(path string) (os.FileInfo, error) {
		if path == "/usr/bin/quickstart-netpolicy" {
			return info, nil
		}
		return os.Stat(path)
	}, run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name != service {
			return nil, errors.New("unexpected launcher")
		}
		if len(args) == 1 && args[0] == "reload" {
			return exec.CommandContext(ctx, "sh", filepath.Join(base, "reload-a")).CombinedOutput()
		}
		return nil, nil
	}}
	module := NewCapabilityActionModule(store)
	ctx := withLANScope(context.Background(), scopes[0])
	request := &models.CapabilityActionRequest{CapabilityKey: "device_speed_limit", Action: "enable"}
	plan, err := module.Plan(ctx, request)
	if err != nil || plan.Result.Error != nil || !plan.Result.Plan.CanApply {
		t.Fatalf("connection plan: %+v %v", plan, err)
	}
	request.Confirm = true
	request.ExpectedVersion = plan.Result.Plan.Version
	result, err := module.Apply(ctx, request)
	if err != nil || result.Result.Error != nil || result.Result.Current.State != "available" {
		t.Fatalf("connection apply: %+v %v", result, err)
	}
}

func TestMultiLANLiveAuxiliaryScheduler(t *testing.T) {
	base := auxiliaryLiveBase(t)
	inventory, scopes := auxiliaryLiveInventory(t)
	deviceRestrictionConfigDir = func() string { return filepath.Join(base, "config") }
	apply := func(ctx context.Context) error {
		out, err := exec.CommandContext(ctx, "sh", filepath.Join(base, "load-firewall")).CombinedOutput()
		if err != nil {
			return fmt.Errorf("isolated fw4: %w: %s", err, out)
		}
		return nil
	}
	deviceAccessRestrictionApply = apply
	deviceRestrictionApply = func(ctx context.Context, configs []string) error { return nil }
	store := newJSONDeviceGroupStore(filepath.Join(base, "groups.json"))
	now := time.Now()
	start := (now.Hour()*60 + now.Minute() + 1) % 1440
	end := (start + 1) % 1440
	schedule := &models.GroupSchedule{ID: "window", Enabled: true, Days: []int{0, 1, 2, 3, 4, 5, 6}, StartMinute: start, EndMinute: end, Action: "block"}
	const id = "mac:02:00:00:00:00:55"
	for index, scope := range scopes {
		policy := &models.GroupPolicy{}
		if index == 0 {
			policy.Schedules = []*models.GroupSchedule{schedule}
		}
		if _, err := store.Mutate(withLANScope(context.Background(), scope), &models.DeviceGroupMutationRequest{Action: "upsert_group", Group: &models.DeviceGroup{ID: "scheduled", Name: "scheduled", Members: []string{id}, Policy: policy}}, ""); err != nil {
			t.Fatal(err)
		}
	}
	policy := newDevicePolicyModuleForTest(&systemDevicePolicyStore{inventory: inventory})
	module := NewDeviceGroupModule(store, &defaultGroupPolicyApplier{policy: policy})
	module.inventory = inventory
	module.AttachLANScopes(&LANScopeModule{reader: fakeLANScopeReader{scopes: scopes}, now: time.Now})
	ready, _ := json.Marshal(map[string]any{"startMinute": start, "endMinute": end, "startedAt": now.UTC().Format(time.RFC3339Nano)})
	if err := os.WriteFile(filepath.Join(base, "scheduler.ready"), ready, 0600); err != nil {
		t.Fatal(err)
	}
	select {}
}
