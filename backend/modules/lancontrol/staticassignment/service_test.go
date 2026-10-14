package staticassignment

import "testing"

func TestHasDuplicateIPConflict(t *testing.T) {
	t.Parallel()
	hosts := []HostRecord{
		{SectionName: "same", MAC: "AA:BB:CC:DD:EE:01", IP: "192.168.100.20"},
		{SectionName: "other", MAC: "AA:BB:CC:DD:EE:02", IP: "192.168.100.30"},
	}
	if HasDuplicateIPConflict(Input{Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:01", AssignedIP: "192.168.100.20", BindIP: true}, hosts) {
		t.Fatal("same MAC should not conflict with its own address")
	}
	if !HasDuplicateIPConflict(Input{Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:03", AssignedIP: "192.168.100.30", BindIP: true}, hosts) {
		t.Fatal("different MAC should conflict on the same address")
	}
	if HasDuplicateIPConflict(Input{Action: "delete", AssignedMAC: "AA:BB:CC:DD:EE:03", AssignedIP: "192.168.100.30", BindIP: true}, hosts) {
		t.Fatal("delete should not conflict")
	}
	if HasDuplicateIPConflict(Input{Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:03", AssignedIP: "192.168.100.30", BindIP: false}, hosts) {
		t.Fatal("unbound address should not conflict")
	}
}
