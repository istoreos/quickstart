package service

import (
	"strings"
	"testing"
)

func TestNormalizeStaticAssignmentInputAcceptsPortableHostname(t *testing.T) {
	t.Parallel()
	got, err := normalizeStaticAssignmentInput(StaticAssignmentWriteInput{
		Action: "add", AssignedMAC: "aa:bb:cc:dd:ee:ff", AssignedIP: "192.168.100.20",
		BindIP: true, Hostname: "Living-Room-TV", TagName: "t_auto_c0a86401", TagTitle: "旁路由",
	})
	if err != nil {
		t.Fatalf("normalize input: %v", err)
	}
	if got.AssignedMAC != "AA:BB:CC:DD:EE:FF" || got.Hostname != "living-room-tv" || got.AssignedIP != "192.168.100.20" {
		t.Fatalf("normalized input = %+v", got)
	}
}

func TestNormalizeStaticAssignmentInputRejectsUnsafeHostnames(t *testing.T) {
	t.Parallel()
	invalid := []string{
		"客厅电视", "living room", "living_room", "-router", "router-", "router.local",
		"router'", "router\\", "router\nnext", strings.Repeat("a", 64), "😀",
	}
	for _, hostname := range invalid {
		hostname := hostname
		t.Run(hostname, func(t *testing.T) {
			t.Parallel()
			_, err := normalizeStaticAssignmentInput(StaticAssignmentWriteInput{
				Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:FF", Hostname: hostname,
			})
			if err == nil {
				t.Fatalf("hostname %q should be rejected", hostname)
			}
		})
	}
}

func TestNormalizeStaticAssignmentInputRejectsInvalidNetworkFields(t *testing.T) {
	t.Parallel()
	tests := []StaticAssignmentWriteInput{
		{Action: "add", AssignedMAC: "not-a-mac"},
		{Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:FF", AssignedIP: "2001:db8::1"},
		{Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:FF", BindIP: true},
		{Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:FF", TagName: "route'; reboot"},
		{Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:FF", TagTitle: "route\nnext"},
		{Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:FF", TagTitle: "route'next"},
	}
	for index, input := range tests {
		if _, err := normalizeStaticAssignmentInput(input); err == nil {
			t.Fatalf("case %d should be rejected: %+v", index, input)
		}
	}
}

func TestNormalizeStaticAssignmentInputAllowsDeletingLegacyInvalidValues(t *testing.T) {
	t.Parallel()
	got, err := normalizeStaticAssignmentInput(StaticAssignmentWriteInput{
		Action: "delete", AssignedMAC: "aa:bb:cc:dd:ee:ff", Hostname: "旧的 中文 名称", TagName: "legacy bad tag",
	})
	if err != nil {
		t.Fatalf("delete legacy rule: %v", err)
	}
	if got.AssignedMAC != "AA:BB:CC:DD:EE:FF" || got.Hostname != "" || got.TagName != "" {
		t.Fatalf("normalized delete = %+v", got)
	}
}
