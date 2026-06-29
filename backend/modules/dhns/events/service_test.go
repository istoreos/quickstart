package events

import (
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestShouldTriggerIfaceEvent(t *testing.T) {
	tests := []struct {
		name string
		evt  models.DHNSChangeRequest
		want bool
	}{
		{
			name: "iface up event",
			evt:  models.DHNSChangeRequest{Action: "ifaceEvent", Params: []string{"up", "wan"}},
			want: true,
		},
		{
			name: "iface down event",
			evt:  models.DHNSChangeRequest{Action: "ifaceEvent", Params: []string{"down", "wan"}},
			want: true,
		},
		{
			name: "uci change",
			evt:  models.DHNSChangeRequest{Action: "uciChange"},
			want: true,
		},
		{
			name: "iface event with invalid direction",
			evt:  models.DHNSChangeRequest{Action: "ifaceEvent", Params: []string{"sideways", "wan"}},
			want: false,
		},
		{
			name: "iface event with missing interface",
			evt:  models.DHNSChangeRequest{Action: "ifaceEvent", Params: []string{"up"}},
			want: false,
		},
		{
			name: "unknown action",
			evt:  models.DHNSChangeRequest{Action: "unknown"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldTriggerIfaceEvent(tt.evt)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
