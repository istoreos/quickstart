package networkbasics

import (
	"reflect"
	"testing"
)

func TestBuildLanDHCPRangeCommands(t *testing.T) {
	got := BuildLanDHCPRangeCommands(100, 150)
	want := []string{
		"uci set dhcp.lan.start=100",
		"uci set dhcp.lan.limit=150",
		"uci set dhcp.lan.leasetime=12h",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBuildWANModePendingConfigs(t *testing.T) {
	tests := []struct {
		name          string
		includeMasq   bool
		enableLanDhcp bool
		want          []string
	}{
		{
			name:          "masq and lan dhcp",
			includeMasq:   true,
			enableLanDhcp: true,
			want:          []string{"firewall", "dhcp", "network"},
		},
		{
			name:          "network only",
			includeMasq:   false,
			enableLanDhcp: false,
			want:          []string{"network"},
		},
		{
			name:          "lan dhcp",
			includeMasq:   false,
			enableLanDhcp: true,
			want:          []string{"dhcp", "network"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildWANModePendingConfigs(tt.includeMasq, tt.enableLanDhcp)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %#v, got %#v", tt.want, got)
			}
		})
	}
}
