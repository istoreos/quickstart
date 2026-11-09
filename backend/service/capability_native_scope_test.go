package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func scopedCapabilityFixture(t *testing.T) (*systemCapabilityActionStore, context.Context, *[]string) {
	t.Helper()
	dir := t.TempDir()
	indexRaw, err := os.ReadFile("/sys/class/net/lo/ifindex")
	if err != nil {
		t.Skip("kernel loopback required")
	}
	index, _ := strconv.Atoi(strings.TrimSpace(string(indexRaw)))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data any
		switch r.URL.Path {
		case "/api/v1/system/capabilities":
			data = map[string]any{"product": "quickstart-netpolicy", "interfaceVersion": "1", "engine": "quickstart-native", "scopeContextVersion": 1}
		case "/api/v1/system/health":
			data = map[string]any{"state": "healthy", "interface": "lo", "context": map[string]any{"networkId": "office", "instanceId": "office", "interfaceIndex": index, "epoch": "fixture"}}
		default:
			t.Errorf("unexpected native call %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": data})
	}))
	t.Cleanup(server.Close)
	port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")
	config := fmt.Sprintf("config quickstart-netpolicy 'main'\n option enabled '1'\n option network 'lan'\n option interface 'eth0'\n option port '8765'\n option data_dir '%s/main'\nconfig quickstart-netpolicy 'office'\n option enabled '0'\n option network 'office'\n option interface 'lo'\n option port '%s'\n option data_dir '%s/office'\n", dir, port, dir)
	if err := os.WriteFile(filepath.Join(dir, "quickstart-netpolicy"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	service := filepath.Join(dir, "init")
	if err := os.WriteFile(service, []byte("# "+nativeScopeContract+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(service)
	if err != nil {
		t.Fatal(err)
	}
	calls := []string{}
	store := &systemCapabilityActionStore{configDir: dir, servicePath: service, stat: func(path string) (os.FileInfo, error) {
		if path == "/usr/bin/quickstart-netpolicy" {
			return info, nil
		}
		return os.Stat(path)
	}, run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != service {
			return nil, errors.New("unexpected service")
		}
		calls = append(calls, strings.Join(args, " "))
		return nil, nil
	}}
	ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: "office", Device: "lo", State: "up", Version: "binding"})
	return store, ctx, &calls
}

func TestMultiLANComponentConnectionIsSelectedAndVersioned(t *testing.T) {
	store, ctx, calls := scopedCapabilityFixture(t)
	module := NewCapabilityActionModule(store)
	request := &models.CapabilityActionRequest{CapabilityKey: "device_speed_limit", Action: "enable"}
	plan, err := module.Plan(ctx, request)
	if err != nil || !plan.Result.Plan.CanApply || plan.Result.Plan.Current.State != "disabled" || len(plan.Result.Plan.AffectedNetworks) != 2 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	if len(*calls) != 0 {
		t.Fatal("plan started a service")
	}
	request.Confirm = true
	request.ExpectedVersion = plan.Result.Plan.Version
	applied, err := module.Apply(ctx, request)
	if err != nil || applied.Result.Error != nil || applied.Result.Current.State != "available" {
		t.Fatalf("apply=%+v err=%v", applied, err)
	}
	tree := uci.NewTree(store.configDir)
	if err := tree.LoadConfig("quickstart-netpolicy", true); err != nil {
		t.Fatal(err)
	}
	for section, want := range map[string]string{"main": "1", "office": "1"} {
		flag, _ := tree.GetLast("quickstart-netpolicy", section, "enabled")
		if flag != want {
			t.Fatalf("%s=%s", section, flag)
		}
	}
	device, _ := tree.GetLast("quickstart-netpolicy", "main", "interface")
	if device != "eth0" {
		t.Fatal("changed other network's device")
	}
	components := module.Components(ctx, nativePolicyProviderName)
	if components.Result.Components[0].Installed == nil || !*components.Result.Components[0].Installed || components.Result.Components[0].NetworkID != "office" {
		t.Fatal("installation and connection were conflated")
	}
	for _, call := range *calls {
		if call == "stop" {
			t.Fatal("stopped every network")
		}
	}
}

func TestMultiLANComponentRejectsStalePlanAndOldService(t *testing.T) {
	store, ctx, calls := scopedCapabilityFixture(t)
	module := NewCapabilityActionModule(store)
	request := &models.CapabilityActionRequest{CapabilityKey: "device_speed_limit", Action: "enable"}
	plan, err := module.Plan(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	request.Confirm = true
	request.ExpectedVersion = plan.Result.Plan.Version
	path := filepath.Join(store.configDir, "quickstart-netpolicy")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, []byte("\n# changed after preview\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	response, err := module.Apply(ctx, request)
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "conflict" || len(*calls) != 0 {
		t.Fatalf("stale action wrote config: %+v %v", response, err)
	}
	if err := os.WriteFile(store.servicePath, []byte("# original single-instance service\n"), 0600); err != nil {
		t.Fatal(err)
	}
	status, err := store.Read(ctx, "device_speed_limit")
	if err != nil || status.State != "unsupported" || status.Reason != "multi_lan_service_upgrade_required" {
		t.Fatalf("old service accepted multi LAN: %+v %v", status, err)
	}
}

func TestMultiLANComponentFailureRestoresConfigWithoutStoppingOthers(t *testing.T) {
	store, ctx, _ := scopedCapabilityFixture(t)
	path := filepath.Join(store.configDir, "quickstart-netpolicy")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reloads := 0
	store.run = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) == 1 && args[0] == "stop" {
			t.Fatal("stopped every network")
		}
		if len(args) == 1 && args[0] == "reload" {
			reloads++
			if reloads == 1 {
				return nil, errors.New("injected reload failure")
			}
		}
		return nil, nil
	}
	if err := store.Enable(ctx, "device_speed_limit"); err == nil {
		t.Fatal("failed connection reported success")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) || reloads != 2 {
		t.Fatalf("configuration recovery failed: reloads=%d err=%v", reloads, err)
	}
}
