package downloadservices

import "testing"

func TestMapStatus(t *testing.T) {
	tests := []struct {
		name      string
		installed bool
		running   bool
		want      string
	}{
		{name: "not installed", installed: false, running: false, want: "not installed"},
		{name: "running", installed: true, running: true, want: "running"},
		{name: "stopped", installed: true, running: false, want: "stopped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapStatus(tt.installed, tt.running)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestBuildStatusResponse(t *testing.T) {
	resp := BuildStatusResponse(
		&Aria2Snapshot{
			Status:       "running",
			ConfigPath:   "/etc/aria2",
			DownloadPath: "/mnt/aria2",
			RPCPort:      6800,
			RPCToken:     "token",
			WebPath:      "/ariang",
		},
		&QbittorrentSnapshot{
			Status:       "stopped",
			ConfigPath:   "/etc/qbit",
			DownloadPath: "/mnt/qbit",
			WebPath:      ":8080",
		},
		&TransmissionSnapshot{
			Status:       "not installed",
			ConfigPath:   "/etc/transmission",
			DownloadPath: "/mnt/trans",
			WebPath:      ":9091",
		},
	)

	if resp == nil || resp.Result == nil {
		t.Fatalf("expected response model, got %#v", resp)
	}
	if resp.Result.Aria2 == nil || resp.Result.Aria2.Status != "running" || resp.Result.Aria2.RPCPort != 6800 || resp.Result.Aria2.WebPath != "/ariang" {
		t.Fatalf("unexpected aria2 result: %#v", resp.Result.Aria2)
	}
	if resp.Result.Qbittorrent == nil || resp.Result.Qbittorrent.Status != "stopped" || resp.Result.Qbittorrent.WebPath != ":8080" {
		t.Fatalf("unexpected qbittorrent result: %#v", resp.Result.Qbittorrent)
	}
	if resp.Result.Transmission == nil || resp.Result.Transmission.Status != "not installed" || resp.Result.Transmission.WebPath != ":9091" {
		t.Fatalf("unexpected transmission result: %#v", resp.Result.Transmission)
	}
}
