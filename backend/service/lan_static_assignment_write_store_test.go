package service

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
)

var lanStaticAssignmentWriteTestMu sync.Mutex

func TestDefaultStaticAssignmentWriteStoreRejectsDuplicateIPConflict(t *testing.T) {
	lanStaticAssignmentWriteTestMu.Lock()
	defer lanStaticAssignmentWriteTestMu.Unlock()

	originalLoadConfig := lanStaticAssignmentWriteLoadConfig
	originalGetSections := lanStaticAssignmentWriteGetSections
	originalGetLast := lanStaticAssignmentWriteGetLast
	originalBatchRun := lanStaticAssignmentWriteBatchRun
	originalCommitAndApply := lanStaticAssignmentWriteCommitAndApply
	t.Cleanup(func() {
		lanStaticAssignmentWriteLoadConfig = originalLoadConfig
		lanStaticAssignmentWriteGetSections = originalGetSections
		lanStaticAssignmentWriteGetLast = originalGetLast
		lanStaticAssignmentWriteBatchRun = originalBatchRun
		lanStaticAssignmentWriteCommitAndApply = originalCommitAndApply
	})

	lanStaticAssignmentWriteLoadConfig = func(config string, overwrite bool) error {
		return nil
	}
	lanStaticAssignmentWriteGetSections = func(config, stype string) ([]string, bool) {
		return []string{"cfg01"}, true
	}
	lanStaticAssignmentWriteGetLast = func(config, section, option string) (string, bool) {
		switch option {
		case "mac":
			return "AA:BB:CC:DD:EE:09", true
		case "ip":
			return "192.168.100.12", true
		default:
			return "", false
		}
	}
	lanStaticAssignmentWriteBatchRun = func(ctx context.Context, cmdList []string, timeout int) error {
		t.Fatalf("BatchRun should not be called on duplicate IP conflict")
		return nil
	}
	lanStaticAssignmentWriteCommitAndApply = func(ctx context.Context, configs []string) error {
		t.Fatalf("UciCommitAndApply should not be called on duplicate IP conflict")
		return nil
	}

	store := NewDefaultStaticAssignmentWriteStore()
	err := store.ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{
		Action:      "add",
		AssignedMAC: "AA:BB:CC:DD:EE:03",
		AssignedIP:  "192.168.100.12",
		BindIP:      true,
	})
	if err == nil || err.Error() != "ip is already in use" {
		t.Fatalf("err = %v, want ip is already in use", err)
	}
}

func TestDefaultStaticAssignmentWriteStoreIgnoresDuplicateIPWhenBindIPDisabled(t *testing.T) {
	lanStaticAssignmentWriteTestMu.Lock()
	defer lanStaticAssignmentWriteTestMu.Unlock()

	originalLoadConfig := lanStaticAssignmentWriteLoadConfig
	originalGetSections := lanStaticAssignmentWriteGetSections
	originalGetLast := lanStaticAssignmentWriteGetLast
	originalBatchRun := lanStaticAssignmentWriteBatchRun
	originalCommitAndApply := lanStaticAssignmentWriteCommitAndApply
	t.Cleanup(func() {
		lanStaticAssignmentWriteLoadConfig = originalLoadConfig
		lanStaticAssignmentWriteGetSections = originalGetSections
		lanStaticAssignmentWriteGetLast = originalGetLast
		lanStaticAssignmentWriteBatchRun = originalBatchRun
		lanStaticAssignmentWriteCommitAndApply = originalCommitAndApply
	})

	lanStaticAssignmentWriteLoadConfig = func(config string, overwrite bool) error {
		return nil
	}
	lanStaticAssignmentWriteGetSections = func(config, stype string) ([]string, bool) {
		return []string{"cfg01"}, true
	}
	lanStaticAssignmentWriteGetLast = func(config, section, option string) (string, bool) {
		switch option {
		case "mac":
			return "AA:BB:CC:DD:EE:09", true
		case "ip":
			return "192.168.100.12", true
		default:
			return "", false
		}
	}

	var gotCommands []string
	lanStaticAssignmentWriteBatchRun = func(ctx context.Context, cmdList []string, timeout int) error {
		gotCommands = append(gotCommands, cmdList...)
		return nil
	}
	lanStaticAssignmentWriteCommitAndApply = func(ctx context.Context, configs []string) error {
		return nil
	}

	store := NewDefaultStaticAssignmentWriteStore()
	err := store.ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{
		Action:      "add",
		AssignedMAC: "AA:BB:CC:DD:EE:03",
		AssignedIP:  "192.168.100.12",
		BindIP:      false,
	})
	if err != nil {
		t.Fatalf("ApplyStaticAssignment returned error: %v", err)
	}
	if len(gotCommands) == 0 {
		t.Fatal("expected command plan to be executed")
	}
	for _, command := range gotCommands {
		if command == "uci set dhcp.@host[-1].ip='192.168.100.12'" {
			t.Fatalf("unexpected ip write command: %q", command)
		}
	}
}

func TestDefaultStaticAssignmentWriteStoreAppliesPlannedCommands(t *testing.T) {
	lanStaticAssignmentWriteTestMu.Lock()
	defer lanStaticAssignmentWriteTestMu.Unlock()

	originalLoadConfig := lanStaticAssignmentWriteLoadConfig
	originalGetSections := lanStaticAssignmentWriteGetSections
	originalGetLast := lanStaticAssignmentWriteGetLast
	originalBatchRun := lanStaticAssignmentWriteBatchRun
	originalCommitAndApply := lanStaticAssignmentWriteCommitAndApply
	t.Cleanup(func() {
		lanStaticAssignmentWriteLoadConfig = originalLoadConfig
		lanStaticAssignmentWriteGetSections = originalGetSections
		lanStaticAssignmentWriteGetLast = originalGetLast
		lanStaticAssignmentWriteBatchRun = originalBatchRun
		lanStaticAssignmentWriteCommitAndApply = originalCommitAndApply
	})

	lanStaticAssignmentWriteLoadConfig = func(config string, overwrite bool) error {
		return nil
	}
	lanStaticAssignmentWriteGetSections = func(config, stype string) ([]string, bool) {
		return []string{"cfg01"}, true
	}
	lanStaticAssignmentWriteGetLast = func(config, section, option string) (string, bool) {
		switch option {
		case "mac":
			return "AA:BB:CC:DD:EE:99", true
		case "ip":
			return "192.168.100.50", true
		default:
			return "", false
		}
	}

	var gotCommands []string
	lanStaticAssignmentWriteBatchRun = func(ctx context.Context, cmdList []string, timeout int) error {
		gotCommands = append(gotCommands, cmdList...)
		return nil
	}
	var committed []string
	lanStaticAssignmentWriteCommitAndApply = func(ctx context.Context, configs []string) error {
		committed = append(committed, configs...)
		return nil
	}

	store := NewDefaultStaticAssignmentWriteStore()
	err := store.ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{
		Action:      "add",
		AssignedMAC: "AA:BB:CC:DD:EE:03",
		AssignedIP:  "192.168.100.12",
		BindIP:      true,
		Hostname:    "printer",
		TagName:     "guest",
		TagTitle:    "Guest",
	})
	if err != nil {
		t.Fatalf("ApplyStaticAssignment returned error: %v", err)
	}
	wantCommands := []string{
		"uci add dhcp host",
		"uci set dhcp.@host[-1].enabled='1'",
		"uci set dhcp.@host[-1].mac='AA:BB:CC:DD:EE:03'",
		"uci set dhcp.@host[-1].tag='guest'",
		"uci set dhcp.@host[-1].tag_title='Guest'",
		"uci set dhcp.@host[-1].name='printer'",
		"uci set dhcp.@host[-1].ip='192.168.100.12'",
	}
	if !reflect.DeepEqual(gotCommands, wantCommands) {
		t.Fatalf("commands = %#v, want %#v", gotCommands, wantCommands)
	}
	if len(committed) != 1 || committed[0] != "dhcp" {
		t.Fatalf("committed = %+v, want [dhcp]", committed)
	}
}

func TestDefaultStaticAssignmentWriteStorePropagatesBatchRunError(t *testing.T) {
	lanStaticAssignmentWriteTestMu.Lock()
	defer lanStaticAssignmentWriteTestMu.Unlock()

	originalLoadConfig := lanStaticAssignmentWriteLoadConfig
	originalGetSections := lanStaticAssignmentWriteGetSections
	originalGetLast := lanStaticAssignmentWriteGetLast
	originalBatchRun := lanStaticAssignmentWriteBatchRun
	originalCommitAndApply := lanStaticAssignmentWriteCommitAndApply
	t.Cleanup(func() {
		lanStaticAssignmentWriteLoadConfig = originalLoadConfig
		lanStaticAssignmentWriteGetSections = originalGetSections
		lanStaticAssignmentWriteGetLast = originalGetLast
		lanStaticAssignmentWriteBatchRun = originalBatchRun
		lanStaticAssignmentWriteCommitAndApply = originalCommitAndApply
	})

	lanStaticAssignmentWriteLoadConfig = func(config string, overwrite bool) error {
		return nil
	}
	lanStaticAssignmentWriteGetSections = func(config, stype string) ([]string, bool) {
		return nil, true
	}
	lanStaticAssignmentWriteGetLast = func(config, section, option string) (string, bool) {
		return "", false
	}
	lanStaticAssignmentWriteBatchRun = func(ctx context.Context, cmdList []string, timeout int) error {
		return errors.New("batch failed")
	}
	lanStaticAssignmentWriteCommitAndApply = func(ctx context.Context, configs []string) error {
		t.Fatalf("UciCommitAndApply should not be called after batch failure")
		return nil
	}

	store := NewDefaultStaticAssignmentWriteStore()
	err := store.ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{
		Action:      "delete",
		AssignedMAC: "AA:BB:CC:DD:EE:03",
	})
	if err == nil || err.Error() != "batch failed" {
		t.Fatalf("err = %v, want batch failed", err)
	}
}
