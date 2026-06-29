package staticassignment

import (
	"reflect"
	"testing"
)

func TestBuildCommandsDeletesExistingMACBeforeAdd(t *testing.T) {
	t.Parallel()

	got := BuildCommands(
		Input{
			Action:      "add",
			AssignedMAC: "AA:BB:CC:DD:EE:FF",
			AssignedIP:  "192.168.100.10",
			BindIP:      true,
			Hostname:    "printer",
			TagName:     "guest",
			TagTitle:    "Guest",
		},
		[]HostRecord{
			{SectionName: "cfg01", MAC: "AA:BB:CC:DD:EE:FF"},
		},
		false,
	)

	want := []string{
		"uci del dhcp.cfg01",
		"uci commit dhcp",
		"uci add dhcp host",
		"uci set dhcp.@host[-1].enabled='1'",
		"uci set dhcp.@host[-1].mac='AA:BB:CC:DD:EE:FF'",
		"uci set dhcp.@host[-1].tag='guest'",
		"uci set dhcp.@host[-1].tag_title='Guest'",
		"uci set dhcp.@host[-1].name='printer'",
		"uci set dhcp.@host[-1].ip='192.168.100.10'",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %#v, want %#v", got, want)
	}
}

func TestBuildCommandsOmitsIPWhenBindIPDisabled(t *testing.T) {
	t.Parallel()

	got := BuildCommands(
		Input{
			Action:      "add",
			AssignedMAC: "AA:BB:CC:DD:EE:01",
			AssignedIP:  "192.168.100.11",
			BindIP:      false,
		},
		nil,
		false,
	)

	for _, command := range got {
		if command == "uci set dhcp.@host[-1].ip='192.168.100.11'" {
			t.Fatalf("unexpected ip write command: %q", command)
		}
	}
}

func TestBuildCommandsIncludesAutoTagMaterialization(t *testing.T) {
	t.Parallel()

	got := BuildCommands(
		Input{
			Action:      "add",
			AssignedMAC: "AA:BB:CC:DD:EE:02",
			TagName:     "t_auto_lan2",
		},
		nil,
		true,
	)

	wantPrefix := []string{
		"uci set dhcp.t_auto_lan2=tag",
		"uci add dhcp host",
	}
	if !reflect.DeepEqual(got[:2], wantPrefix) {
		t.Fatalf("command prefix = %#v, want %#v", got[:2], wantPrefix)
	}
}

func TestBuildCommandsPreservesDeleteOnlyFlow(t *testing.T) {
	t.Parallel()

	got := BuildCommands(
		Input{
			Action:      "delete",
			AssignedMAC: "AA:BB:CC:DD:EE:AA",
		},
		[]HostRecord{{SectionName: "cfg09", MAC: "AA:BB:CC:DD:EE:AA"}},
		false,
	)
	want := []string{
		"uci del dhcp.cfg09",
		"uci commit dhcp",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %#v, want %#v", got, want)
	}
}

func TestHasDuplicateIPConflict(t *testing.T) {
	t.Parallel()

	hosts := []HostRecord{
		{MAC: "AA:BB:CC:DD:EE:09", IP: "192.168.100.12"},
	}
	tests := []struct {
		name  string
		input Input
		want  bool
	}{
		{
			name: "conflicting bound ip",
			input: Input{
				Action:      "add",
				AssignedMAC: "AA:BB:CC:DD:EE:03",
				AssignedIP:  "192.168.100.12",
				BindIP:      true,
			},
			want: true,
		},
		{
			name: "bind ip disabled",
			input: Input{
				Action:      "add",
				AssignedMAC: "AA:BB:CC:DD:EE:03",
				AssignedIP:  "192.168.100.12",
				BindIP:      false,
			},
		},
		{
			name: "delete",
			input: Input{
				Action:      "delete",
				AssignedMAC: "AA:BB:CC:DD:EE:03",
				AssignedIP:  "192.168.100.12",
				BindIP:      true,
			},
		},
		{
			name: "same mac",
			input: Input{
				Action:      "add",
				AssignedMAC: "AA:BB:CC:DD:EE:09",
				AssignedIP:  "192.168.100.12",
				BindIP:      true,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := HasDuplicateIPConflict(tt.input, hosts)
			if got != tt.want {
				t.Fatalf("HasDuplicateIPConflict() = %v, want %v", got, tt.want)
			}
		})
	}
}
