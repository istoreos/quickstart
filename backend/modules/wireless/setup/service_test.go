package setup

import (
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestPrepareQuickSetupRequestRequiresFields(t *testing.T) {
	tests := []struct {
		name string
		req  *models.WirelessQuickSetupRequest
	}{
		{
			name: "missing 2g iface",
			req: &models.WirelessQuickSetupRequest{
				Wifi5g: validIface("wifi5g", "ssid5g", validKey(8)),
			},
		},
		{
			name: "missing 5g iface",
			req: &models.WirelessQuickSetupRequest{
				Wifi2g: validIface("wifi2g", "ssid2g", validKey(8)),
			},
		},
		{
			name: "missing 2g ssid",
			req: validQuickSetupRequest(func(req *models.WirelessQuickSetupRequest) {
				req.Wifi2g.Ssid = ""
			}),
		},
		{
			name: "missing 5g ssid",
			req: validQuickSetupRequest(func(req *models.WirelessQuickSetupRequest) {
				req.Wifi5g.Ssid = ""
			}),
		},
		{
			name: "missing 2g key",
			req: validQuickSetupRequest(func(req *models.WirelessQuickSetupRequest) {
				req.Wifi2g.Key = ""
			}),
		},
		{
			name: "missing 5g key",
			req: validQuickSetupRequest(func(req *models.WirelessQuickSetupRequest) {
				req.Wifi5g.Key = ""
			}),
		},
		{
			name: "missing 2g iface name",
			req: validQuickSetupRequest(func(req *models.WirelessQuickSetupRequest) {
				req.Wifi2g.IfaceName = ""
			}),
		},
		{
			name: "missing 5g iface name",
			req: validQuickSetupRequest(func(req *models.WirelessQuickSetupRequest) {
				req.Wifi5g.IfaceName = ""
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := PrepareQuickSetupRequest(tt.req); err == nil || err.Error() != "Invalid params" {
				t.Fatalf("PrepareQuickSetupRequest() error = %v, want Invalid params", err)
			}
		})
	}
}

func TestPrepareQuickSetupRequestValidatesPasswordLength(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*models.WirelessQuickSetupRequest)
		wantErr string
	}{
		{
			name: "2g length 7",
			mutate: func(req *models.WirelessQuickSetupRequest) {
				req.Wifi2g.Key = validKey(7)
			},
			wantErr: "Invalid 2g password",
		},
		{
			name: "2g length 8",
			mutate: func(req *models.WirelessQuickSetupRequest) {
				req.Wifi2g.Key = validKey(8)
			},
		},
		{
			name: "2g length 20",
			mutate: func(req *models.WirelessQuickSetupRequest) {
				req.Wifi2g.Key = validKey(20)
			},
		},
		{
			name: "2g length 21",
			mutate: func(req *models.WirelessQuickSetupRequest) {
				req.Wifi2g.Key = validKey(21)
			},
			wantErr: "Invalid 2g password",
		},
		{
			name: "5g length 7",
			mutate: func(req *models.WirelessQuickSetupRequest) {
				req.Wifi5g.Key = validKey(7)
			},
			wantErr: "Invalid 5g password",
		},
		{
			name: "5g length 8",
			mutate: func(req *models.WirelessQuickSetupRequest) {
				req.Wifi5g.Key = validKey(8)
			},
		},
		{
			name: "5g length 20",
			mutate: func(req *models.WirelessQuickSetupRequest) {
				req.Wifi5g.Key = validKey(20)
			},
		},
		{
			name: "5g length 21",
			mutate: func(req *models.WirelessQuickSetupRequest) {
				req.Wifi5g.Key = validKey(21)
			},
			wantErr: "Invalid 5g password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validQuickSetupRequest(tt.mutate)
			err := PrepareQuickSetupRequest(req)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("PrepareQuickSetupRequest() error = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("PrepareQuickSetupRequest() error = %v, want %s", err, tt.wantErr)
			}
		})
	}
}

func TestPrepareQuickSetupRequestFillsDefaults(t *testing.T) {
	req := validQuickSetupRequest(nil)

	if err := PrepareQuickSetupRequest(req); err != nil {
		t.Fatalf("PrepareQuickSetupRequest() error = %v, want nil", err)
	}

	assertIfaceDefaults(t, req.Wifi2g, "psk-mixed", "auto", "11b/g/n/ax")
	assertIfaceDefaults(t, req.Wifi5g, "psk-mixed", "80", "11a/n/ac/ax")
}

func validQuickSetupRequest(mutate func(*models.WirelessQuickSetupRequest)) *models.WirelessQuickSetupRequest {
	req := &models.WirelessQuickSetupRequest{
		Wifi2g: validIface("wifi2g", "ssid2g", validKey(8)),
		Wifi5g: validIface("wifi5g", "ssid5g", validKey(8)),
	}
	if mutate != nil {
		mutate(req)
	}
	return req
}

func validIface(ifaceName, ssid, key string) *models.WirelessIfaceInfo {
	return &models.WirelessIfaceInfo{
		IfaceName: ifaceName,
		Ssid:      ssid,
		Key:       key,
	}
}

func validKey(length int) string {
	return strings.Repeat("a", length)
}

func assertIfaceDefaults(t *testing.T, iface *models.WirelessIfaceInfo, encryption, htmode, hwmode string) {
	t.Helper()

	if iface.Channel != 0 {
		t.Fatalf("Channel = %d, want 0", iface.Channel)
	}
	if iface.Encryption != encryption {
		t.Fatalf("Encryption = %q, want %q", iface.Encryption, encryption)
	}
	if iface.Htmode != htmode {
		t.Fatalf("Htmode = %q, want %q", iface.Htmode, htmode)
	}
	if iface.Hwmode != hwmode {
		t.Fatalf("Hwmode = %q, want %q", iface.Hwmode, hwmode)
	}
	if iface.Hidden {
		t.Fatalf("Hidden = true, want false")
	}
}
