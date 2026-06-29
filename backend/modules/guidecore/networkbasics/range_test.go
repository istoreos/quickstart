package networkbasics

import "testing"

func TestBuildLANRange(t *testing.T) {
	tests := []struct {
		name      string
		lanIP     string
		start     string
		limit     string
		wantStart string
		wantEnd   string
	}{
		{
			name:      "normal range",
			lanIP:     "192.168.100.1",
			start:     "100",
			limit:     "150",
			wantStart: "192.168.100.100",
			wantEnd:   "192.168.100.249",
		},
		{
			name:      "invalid lan ip",
			lanIP:     "192.168.100",
			start:     "100",
			limit:     "150",
			wantStart: "",
			wantEnd:   "",
		},
		{
			name:      "legacy parse fallback",
			lanIP:     "192.168.100.1",
			start:     "bad",
			limit:     "150",
			wantStart: "192.168.100.bad",
			wantEnd:   "192.168.100.149",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := BuildLANRange(tt.lanIP, tt.start, tt.limit)
			if gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Fatalf("expected (%q, %q), got (%q, %q)", tt.wantStart, tt.wantEnd, gotStart, gotEnd)
			}
		})
	}
}
