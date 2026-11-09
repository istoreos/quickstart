package service

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

// Browser integration runs real HTTP routes and persistent profile services.
// All data is disposable; the allowlist prevents invoking system configuration.
func TestMultiLANBrowserHTTP(t *testing.T) {
	base := os.Getenv("QUICKSTART_MULTI_LAN_BROWSER_BASE")
	if base == "" {
		t.Skip("requires explicit browser integration harness")
	}
	if !filepath.IsAbs(base) {
		t.Fatal("requires an absolute disposable output directory")
	}
	if err := os.MkdirAll(base, 0700); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{
		ARP:            []deviceInventoryObservation{{NetworkID: "lan1", MAC: "02:00:00:00:00:55", IPv4: "192.0.2.55", Online: true}, {NetworkID: "lan2", MAC: "02:00:00:00:00:55", IPv4: "198.51.100.55", Online: true}},
		NetworkDevices: map[string]string{"lan1": "br-lan1", "lan2": "br-lan2"},
	}}}
	inventory := newDeviceInventoryModuleForTest(source, filepath.Join(base, "history.json"), "browser-test", func() time.Time { return now }, 100)
	backend := &ServiceBackend{deviceInventory: inventory, deviceProfile: NewDeviceProfileModule(inventory), networkAudit: NewNetworkAuditModule(filepath.Join(base, "audit.json"), nil), lanScopes: &LANScopeModule{reader: fakeLANScopeReader{scopes: []*models.LANScope{
		{NetworkID: "lan1", DisplayName: "lan1", Device: "br-lan1", State: "up", Version: "browser-a"},
		{NetworkID: "lan2", DisplayName: "lan2", Device: "br-lan2", State: "up", Version: "browser-b"},
	}}, now: time.Now}}
	allowed := map[string]bool{"/cgi-bin/luci/istore/lanctrl/v2/networks/": true, "/cgi-bin/luci/istore/lanctrl/v2/devices/": true, "/cgi-bin/luci/istore/lanctrl/v2/device-profile/": true}
	listener, err := net.Listen("tcp", "127.0.0.1:19489")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowed[r.URL.Path] || r.Method != "GET" && r.URL.Path != "/cgi-bin/luci/istore/lanctrl/v2/device-profile/" {
			http.Error(w, "outside browser integration scope", 503)
			return
		}
		ctx, err := backend.PrepareLANRequest(r.Context(), r)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var result any
		switch r.URL.Path {
		case "/cgi-bin/luci/istore/lanctrl/v2/networks/":
			result, err = backend.GetLANNetworks(ctx)
		case "/cgi-bin/luci/istore/lanctrl/v2/devices/":
			result, err = backend.GetDeviceInventoryV2(ctx)
		case "/cgi-bin/luci/istore/lanctrl/v2/device-profile/":
			if r.Method == "GET" {
				result, err = backend.GetDeviceProfileV2(ctx, r)
			} else {
				result, err = backend.PostDeviceProfileV2(ctx, r)
			}
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}), ReadHeaderTimeout: 5 * time.Second}
	go server.Serve(listener)
	defer server.Shutdown(context.Background())
	if err := os.WriteFile(filepath.Join(base, "ready"), []byte(listener.Addr().String()), 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(25 * time.Minute)
	for {
		select {
		case <-deadline:
			t.Fatal("browser integration deadline exceeded")
		case <-time.After(250 * time.Millisecond):
			if _, err := os.Stat(filepath.Join(base, "stop")); err == nil {
				return
			}
		}
	}
}
