package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/istoreos/quickstart/backend/models"
)

func TestMultiLANNativeRegistryTargetsVerifiedInstances(t *testing.T) {
	type instance struct {
		mu                    sync.Mutex
		network, device, name string
		index                 int
		mismatch              bool
		writes                int
		snapshot              nativePolicySnapshot
	}
	instances := []*instance{{network: "lan", device: "lo", name: "main"}, {network: "office", device: "eth0", name: "office"}}
	dir := t.TempDir()
	var configuration strings.Builder
	for _, entry := range instances {
		raw, err := os.ReadFile(filepath.Join("/sys/class/net", entry.device, "ifindex"))
		if err != nil {
			t.Skip("two real kernel device indices required")
		}
		entry.index, _ = strconv.Atoi(strings.TrimSpace(string(raw)))
		entry.snapshot = nativePolicySnapshot{SchemaVersion: 1, Policies: []nativeStoredPolicy{}}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			entry.mu.Lock()
			defer entry.mu.Unlock()
			var data any
			switch r.URL.Path {
			case "/api/v1/system/capabilities":
				data = map[string]any{"product": "quickstart-netpolicy", "interfaceVersion": "1", "engine": "quickstart-native", "scopeContextVersion": 1}
			case "/api/v1/system/health":
				network := entry.network
				if entry.mismatch {
					network = "foreign"
				}
				data = map[string]any{"state": "healthy", "interface": entry.device, "context": map[string]any{"networkId": network, "instanceId": entry.name, "epoch": "test-runtime", "interfaceIndex": entry.index}}
			case "/api/v1/policies":
				data = entry.snapshot
			case "/api/v1/policies/plan":
				data = nativePolicyPlan{ID: entry.network, Changes: []nativePolicyChange{}, Warnings: []string{}}
			case "/api/v1/policies/apply":
				var body struct {
					Request nativePolicyMutation `json:"request"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				entry.writes++
				entry.snapshot.Revision++
				entry.snapshot.Policies = []nativeStoredPolicy{}
				for _, input := range body.Request.Policies {
					entry.snapshot.Policies = append(entry.snapshot.Policies, nativeStoredPolicy{Owner: body.Request.Owner, Device: input.Device, RateLimit: input.RateLimit, State: "effective"})
				}
				data = entry.snapshot
			case "/api/v1/policies/verify":
				data = entry.snapshot
			default:
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": data})
		}))
		t.Cleanup(server.Close)
		port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")
		fmt.Fprintf(&configuration, "config quickstart-netpolicy '%s'\n option enabled '1'\n option network '%s'\n option interface '%s'\n option port '%s'\n option data_dir '%s'\n", entry.name, entry.network, entry.device, port, filepath.Join(dir, entry.network))
	}
	if err := os.WriteFile(filepath.Join(dir, "quickstart-netpolicy"), []byte(configuration.String()), 0600); err != nil {
		t.Fatal(err)
	}
	provider := NewNativeRateLimitProvider("http://127.0.0.1:1", nil).(*nativeRateLimitProvider)
	provider.registry = &nativeEngineRegistry{configDir: dir}
	provider.offload = nil
	target := rateLimitTarget{MAC: "02:00:00:00:00:55"}
	var parallel sync.WaitGroup
	failures := make(chan error, len(instances))
	for i, entry := range instances {
		parallel.Add(1)
		go func(entry *instance, rate int64) {
			defer parallel.Done()
			ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: entry.network, Device: entry.device})
			failures <- provider.Apply(ctx, target, models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: rate, DownloadSpeed: rate + 10})
		}(entry, int64(10+i))
	}
	parallel.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}

	for i, entry := range instances {
		ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: entry.network, Device: entry.device})
		observation, err := provider.Inspect(ctx, target)
		if err != nil || observation.Policy.UploadSpeed != int64(10+i) || entry.writes != 1 {
			t.Fatalf("%s observation=%+v writes=%d err=%v", entry.network, observation, entry.writes, err)
		}
	}
	entry := instances[1]
	entry.mu.Lock()
	entry.mismatch = true
	entry.mu.Unlock()
	ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: entry.network, Device: entry.device})
	if err := provider.Apply(ctx, target, models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 1, DownloadSpeed: 1}); err == nil {
		t.Fatal("foreign runtime identity accepted")
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.writes != 1 {
		t.Fatal("binding mismatch produced an engine write")
	}
}

func TestMultiLANNativeRegistryRejectsConflictsAndUnknownNetworks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "quickstart-netpolicy")
	main := "config quickstart-netpolicy 'main'\n option enabled '1'\n option interface 'lo'\n option data_dir '/tmp/qsnp-main'\n"
	other := "config quickstart-netpolicy 'office'\n option enabled '1'\n option network 'office'\n option interface 'eth0'\n option port '8765'\n option data_dir '/tmp/qsnp-office'\n"
	if err := os.WriteFile(path, []byte(main+other), 0600); err != nil {
		t.Fatal(err)
	}
	registry := &nativeEngineRegistry{configDir: dir}
	for _, scope := range []*models.LANScope{{NetworkID: "lan", Device: "lo"}, {NetworkID: "office", Device: "eth0"}, {NetworkID: "missing", Device: "lo"}} {
		if _, err := registry.Resolve(withLANScope(context.Background(), scope)); err == nil {
			t.Fatalf("invalid binding accepted: %+v", scope)
		}
	}
	if err := os.WriteFile(path, []byte(main), 0600); err != nil {
		t.Fatal(err)
	}
	binding, err := registry.Resolve(withLANScope(context.Background(), &models.LANScope{NetworkID: "lan", Device: "lo"}))
	if err != nil || binding.Port != 8765 || binding.InstanceID != "main" {
		t.Fatalf("legacy main binding=%+v err=%v", binding, err)
	}
	if _, err := registry.Resolve(withLANScope(context.Background(), &models.LANScope{NetworkID: "lan", Device: "eth0"})); err == nil {
		t.Fatal("wrong interface accepted")
	}
}

// The hardware runner supplies a task-owned service/configuration. This uses
// the production registry/provider against a real current or legacy engine.
func TestMultiLANNativeLiveBinding(t *testing.T) {
	dir := os.Getenv("QUICKSTART_MULTI_LAN_NATIVE_LIVE_CONFIG")
	if dir == "" {
		t.Skip("explicit isolated native service required")
	}
	network, device := os.Getenv("QUICKSTART_MULTI_LAN_NATIVE_NETWORK"), os.Getenv("QUICKSTART_MULTI_LAN_NATIVE_DEVICE")
	if network == "" || device == "" || !strings.HasPrefix(dir, "/tmp/quickstart-multi-lan/") {
		t.Fatal("invalid isolated test context")
	}
	ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: network, Device: device})
	provider := NewNativeRateLimitProvider("", nil).(*nativeRateLimitProvider)
	provider.registry, provider.offload = &nativeEngineRegistry{configDir: dir}, nil
	if os.Getenv("QUICKSTART_MULTI_LAN_EXPECT_UNSUPPORTED") == "1" {
		err := provider.probe(ctx)
		if err == nil || !strings.Contains(err.Error(), "scope_interface_unsupported") {
			t.Fatalf("expected legacy additional-LAN rejection, got %v", err)
		}
		return
	}
	target := rateLimitTarget{MAC: "02:00:00:00:00:77"}
	if err := provider.Apply(ctx, target, models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 3, DownloadSpeed: 7}); err != nil {
		t.Fatal(err)
	}
	observed, err := provider.Inspect(ctx, target)
	if err != nil || !observed.Verified || observed.Policy.UploadSpeed != 3 || observed.Policy.DownloadSpeed != 7 {
		t.Fatalf("observation=%+v err=%v", observed, err)
	}
	if err := provider.Apply(ctx, target, models.DeviceSpeedPolicy{}); err != nil {
		t.Fatal(err)
	}
	t.Logf("verified real registry/provider read, apply, verify and remove for %s on %s", network, device)
}
