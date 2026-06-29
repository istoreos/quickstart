package service

import "testing"

func TestBuildDhcpGatewayCommandsEnabledWithExplicitGateway(t *testing.T) {
	commands, err := buildDhcpGatewayCommands(
		DhcpGatewayInput{
			DhcpEnabled: true,
			DhcpGateway: "192.168.1.9",
		},
		LanStatusSnapshot{LanAddr: "192.168.1.1"},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []string{
		"uci del dhcp.lan.ignore",
		"uci del dhcp.lan.dhcp_option",
		"uci commit dhcp",
		"uci add_list dhcp.lan.dhcp_option='3,192.168.1.9'",
		"uci add_list dhcp.lan.dhcp_option='6,192.168.1.9'",
	}
	assertCommandList(t, commands, want)
}

func TestBuildDhcpGatewayCommandsDisabled(t *testing.T) {
	commands, err := buildDhcpGatewayCommands(
		DhcpGatewayInput{DhcpEnabled: false},
		LanStatusSnapshot{LanAddr: "192.168.1.1"},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []string{
		"uci del dhcp.lan.dhcp_option",
		"uci set dhcp.lan.ignore=1",
	}
	assertCommandList(t, commands, want)
}

func TestBuildDhcpTagCommandsModifyIncludesHostCleanup(t *testing.T) {
	commands := buildDhcpTagCommands(
		DhcpTagConfigInput{
			Action:     "modify",
			TagName:    "guest",
			TagTitle:   "Guest",
			DhcpOption: []string{"3,192.168.1.9", "6,192.168.1.9"},
		},
		[]dhcpHostTagBinding{
			{SectionName: "host1", TagName: "guest"},
			{SectionName: "host2", TagName: "other"},
			{SectionName: "host3", TagName: "guest"},
		},
	)

	want := []string{
		"del dhcp.guest",
		"set dhcp.guest=tag",
		"set dhcp.guest.tag_title='Guest'",
		"add_list dhcp.guest.dhcp_option='3,192.168.1.9'",
		"add_list dhcp.guest.dhcp_option='6,192.168.1.9'",
		"uci del dhcp.host1",
		"uci del dhcp.host3",
	}
	assertCommandList(t, commands, want)
}

func assertCommandList(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d commands, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected command %q at index %d, got %q", want[i], i, got[i])
		}
	}
}
