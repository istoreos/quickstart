package service

import (
	"context"
	"encoding/json"
	"github.com/istoreos/quickstart/backend/models"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestMultiLANNativeTrafficUsesWANCountersAndRuntime(t *testing.T) {
	for _, tc := range []struct {
		name, epoch string
		age         time.Duration
		want        string
	}{{"ready", "runtime", 0, "ready"}, {"stale", "runtime", 20 * time.Second, "stale"}, {"restarted-between-reads", "other", 0, "partial"}, {"missing-epoch", "", 0, "partial"}, {"future-clock", "runtime", -time.Minute, "partial"}} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var data any
				switch r.URL.Path {
				case "/api/v1/system/capabilities":
					data = map[string]any{"product": "quickstart-netpolicy", "interfaceVersion": "1", "engine": "quickstart-native", "scopeContextVersion": 1}
				case "/api/v1/system/health":
					data = map[string]any{"state": "healthy", "interface": "lo", "context": map[string]any{"networkId": "office", "instanceId": "office", "epoch": "runtime", "interfaceIndex": 1}}
				case "/api/traffic/devices":
					data = map[string]any{"sampledAtMs": time.Now().Add(-tc.age).UnixMilli(), "counterEpoch": tc.epoch, "d": []any{map[string]any{"mac": "02:00:00:00:00:55", "w_tx_b": 120, "w_rx_b": 230, "w_tx_r": 10, "w_rx_r": 20, "l_tx_b": 999999, "l_rx_b": 999999}}}
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": data})
			}))
			defer server.Close()
			provider := NewNativeRateLimitProvider(server.URL, nil).(*nativeRateLimitProvider)
			provider.offload = nil
			inventory := trafficInventoryResponse("mac:02:00:00:00:00:55", "192.0.2.55")
			inventory.Result.Devices[0].Mac = "02:00:00:00:00:55"
			inventory.Result.Devices[0].NetworkID = "office"
			module := newDeviceTrafficModuleForTest(&fakeTrafficInventory{responses: []*models.DeviceInventoryResponse{inventory}}, scopedDeviceTrafficSampler{native: provider}, time.Now)
			response, err := module.Snapshot(inventoryLANContext("office"))
			if err != nil {
				t.Fatal(err)
			}
			item := response.Result.Items[0]
			if item.State != tc.want {
				t.Fatalf("state=%s want=%s", item.State, tc.want)
			}
			if tc.want == "ready" && (item.UploadBytes != 120 || item.DownloadBytes != 230 || item.CounterEpoch != "runtime" || item.NetworkID != "office") {
				t.Fatalf("WAN attribution: %+v", item)
			}
		})
	}
}

func TestMultiLANBackgroundWithoutResolverCannotTargetDefault(t *testing.T) {
	module := NewTrafficInsightsModule(newTrafficInsightsFileStore(t.TempDir()+"/traffic.json"), nil)
	child := module.forContext(inventoryLANContext("office"))
	child.document = emptyTrafficInsightsDocument()
	child.loaded = true
	child.document.Quotas["mac:a"] = &models.TrafficQuota{DeviceID: "mac:a", Enabled: true, LimitBytes: 1, Period: "daily", Action: "notify"}
	called := false
	child.collector = func(context.Context) (*models.DeviceTrafficResponse, error) {
		called = true
		return insightTraffic("mac:a", 100, 0), nil
	}
	child.collectQuotaSample()
	if called {
		t.Fatal("unresolved background scope reached default collector")
	}
}

type scopedTrafficSamplerFunc func(context.Context) lanStatsSnapshot

func (sample scopedTrafficSamplerFunc) Sample(ctx context.Context) lanStatsSnapshot {
	return sample(ctx)
}

func TestMultiLANTrafficAggregateKeepsRowTargetsAndScopedHistory(t *testing.T) {
	now := time.Now()
	const mac = "02:00:00:00:00:55"
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{ARP: []deviceInventoryObservation{{NetworkID: "lan", MAC: mac, IPv4: "192.0.2.55", Online: true}, {NetworkID: "office", MAC: mac, IPv4: "198.51.100.55", Online: true}, {NetworkID: unassignedLAN, MAC: "02:00:00:00:00:66", IPv4: "203.0.113.66"}}}}}, &now, 100)
	sampler := scopedTrafficSamplerFunc(func(ctx context.Context) lanStatsSnapshot {
		up := int64(100)
		if selectedLAN(ctx) == "office" {
			up = 700
		}
		return lanStatsSnapshot{native: map[string]nativeTrafficCounter{mac: {UploadBytes: up}}, epoch: selectedLAN(ctx) + "-runtime", samples: 2, sampledAt: now}
	})
	insights := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(t.TempDir(), "traffic.json")), nil)
	backend := &ServiceBackend{lanScopes: testLANScopeModule(), deviceInventory: inventory, deviceTraffic: newDeviceTrafficModuleForTest(inventory, sampler, func() time.Time { return now }), trafficInsights: insights}
	request := httptest.NewRequest(http.MethodGet, "/cgi-bin/luci/istore/lanctrl/v2/device-traffic/?network=all", nil)
	ctx, err := backend.PrepareLANRequest(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	response, err := backend.GetDeviceTrafficV2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Items) != 3 || len(response.Result.NetworkHealth) != 2 {
		t.Fatalf("aggregate rows: %+v", response.Result)
	}
	for _, item := range response.Result.Items {
		if item.NetworkID == "" {
			if item.State != "unavailable" {
				t.Fatal("unknown row received counters")
			}
			continue
		}
		want := int64(100)
		if item.NetworkID == "office" {
			want = 700
		}
		if item.UploadBytes != want || item.CounterEpoch != item.NetworkID+"-runtime" {
			t.Fatalf("cross-network counter: %+v", item)
		}
	}
	if len(insights.document.Baselines) != 1 || len(insights.forContext(inventoryLANContext("office")).document.Baselines) != 1 {
		t.Fatal("aggregate history wrote into one LAN")
	}
}
