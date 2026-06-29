package service

import (
	"context"
	"errors"
	"sync"
	"testing"
)

var lanSpeedLimitWriteTestMu sync.Mutex

func TestFindSpeedLimitRuleMatchByIP(t *testing.T) {
	t.Parallel()

	match, ok := findSpeedLimitRuleMatchByIP(
		[]SpeedLimitRuleMatch{
			{Config: "eqos", SectionName: "cfg01", MatchIP: "192.168.100.10"},
			{Config: "eqos", SectionName: "cfg02", MatchIP: "192.168.100.11"},
		},
		"192.168.100.11",
	)
	if !ok {
		t.Fatal("expected match")
	}
	if match.SectionName != "cfg02" {
		t.Fatalf("match.SectionName = %q, want %q", match.SectionName, "cfg02")
	}
}

func TestFindSpeedLimitRuleMatchByMAC(t *testing.T) {
	t.Parallel()

	match, ok := findSpeedLimitRuleMatchByMAC(
		[]SpeedLimitRuleMatch{
			{Config: "firewall", SectionName: "cfg11", MatchMAC: "AA:BB:CC:DD:EE:01"},
			{Config: "firewall", SectionName: "cfg12", MatchMAC: "AA:BB:CC:DD:EE:02"},
		},
		"aa:bb:cc:dd:ee:02",
	)
	if !ok {
		t.Fatal("expected match")
	}
	if match.SectionName != "cfg12" {
		t.Fatalf("match.SectionName = %q, want %q", match.SectionName, "cfg12")
	}
}

func TestBuildBlockedRuleNameNormalizesMAC(t *testing.T) {
	t.Parallel()

	got := buildBlockedRuleName("aa:bb:cc:dd:ee:ff")
	if got != "BL_AABBCCDDEEFF" {
		t.Fatalf("buildBlockedRuleName() = %q, want %q", got, "BL_AABBCCDDEEFF")
	}
}

func TestBuildSpeedLimitWritePlanDeletesOldRulesBeforeAdd(t *testing.T) {
	t.Parallel()

	plan := BuildSpeedLimitWritePlan(
		SpeedLimitWriteInput{
			Action:        "add",
			IP:            "192.168.100.20",
			MAC:           "AA:BB:CC:DD:EE:20",
			NetworkAccess: true,
			UploadSpeed:   300,
			DownloadSpeed: 2000,
			Comment:       "tablet",
		},
		[]SpeedLimitRuleMatch{{Config: "eqos", SectionName: "cfg01", MatchIP: "192.168.100.20"}},
		[]SpeedLimitRuleMatch{{Config: "firewall", SectionName: "cfg11", MatchMAC: "AA:BB:CC:DD:EE:20"}},
	)
	if len(plan.DeleteSections) != 2 {
		t.Fatalf("len(DeleteSections) = %d, want 2", len(plan.DeleteSections))
	}
	if plan.DeleteSections[0].SectionName != "cfg01" || plan.DeleteSections[1].SectionName != "cfg11" {
		t.Fatalf("DeleteSections = %+v", plan.DeleteSections)
	}
	if !plan.AddSpeedLimit || plan.AddBlockRule {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestDefaultLanSpeedLimitWriteStoreExecutesDeleteOldEqosBeforeAdd(t *testing.T) {
	lanSpeedLimitWriteTestMu.Lock()
	defer lanSpeedLimitWriteTestMu.Unlock()

	originalExec := lanSpeedLimitWriteExec
	t.Cleanup(func() {
		lanSpeedLimitWriteExec = originalExec
	})

	var got []string
	lanSpeedLimitWriteExec = func(ctx context.Context, commands []string) error {
		_ = ctx
		got = append(got, commands...)
		return nil
	}

	store := NewDefaultLanSpeedLimitWriteStore()
	err := store.ApplyPlan(context.Background(), BuildSpeedLimitWritePlan(
		SpeedLimitWriteInput{
			Action:        "add",
			IP:            "192.168.100.20",
			MAC:           "AA:BB:CC:DD:EE:20",
			NetworkAccess: true,
			UploadSpeed:   300,
			DownloadSpeed: 2000,
			Comment:       "tablet",
		},
		[]SpeedLimitRuleMatch{{Config: "eqos", SectionName: "cfg01", MatchIP: "192.168.100.20"}},
		nil,
	))
	if err != nil {
		t.Fatalf("ApplyPlan returned error: %v", err)
	}
	wantPrefix := []string{
		"uci del eqos.cfg01",
		"uci commit eqos",
		"uci add eqos device",
	}
	for i, want := range wantPrefix {
		if got[i] != want {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want)
		}
	}
}

func TestDefaultLanSpeedLimitWriteStoreExecutesDeleteOldBlockBeforeAdd(t *testing.T) {
	lanSpeedLimitWriteTestMu.Lock()
	defer lanSpeedLimitWriteTestMu.Unlock()

	originalExec := lanSpeedLimitWriteExec
	t.Cleanup(func() {
		lanSpeedLimitWriteExec = originalExec
	})

	var got []string
	lanSpeedLimitWriteExec = func(ctx context.Context, commands []string) error {
		_ = ctx
		got = append(got, commands...)
		return nil
	}

	store := NewDefaultLanSpeedLimitWriteStore()
	err := store.ApplyPlan(context.Background(), BuildSpeedLimitWritePlan(
		SpeedLimitWriteInput{
			Action:        "add",
			MAC:           "AA:BB:CC:DD:EE:20",
			NetworkAccess: false,
		},
		nil,
		[]SpeedLimitRuleMatch{{Config: "firewall", SectionName: "cfg11", MatchMAC: "AA:BB:CC:DD:EE:20"}},
	))
	if err != nil {
		t.Fatalf("ApplyPlan returned error: %v", err)
	}
	wantPrefix := []string{
		"uci del firewall.cfg11",
		"uci commit firewall",
		"uci add firewall rule",
	}
	for i, want := range wantPrefix {
		if got[i] != want {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want)
		}
	}
}

func TestDefaultLanSpeedLimitApplyDelegatesExpectedConfigs(t *testing.T) {
	lanSpeedLimitWriteTestMu.Lock()
	defer lanSpeedLimitWriteTestMu.Unlock()

	originalApply := lanSpeedLimitWriteCommitAndApply
	t.Cleanup(func() {
		lanSpeedLimitWriteCommitAndApply = originalApply
	})

	var got []string
	lanSpeedLimitWriteCommitAndApply = func(ctx context.Context, configs []string) error {
		_ = ctx
		got = append(got, configs...)
		return nil
	}

	apply := NewDefaultLanSpeedLimitApply()
	err := apply.Apply(context.Background(), []string{"eqos", "firewall"})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(got) != 2 || got[0] != "eqos" || got[1] != "firewall" {
		t.Fatalf("got configs = %+v", got)
	}
}

func TestDefaultLanSpeedLimitWriteStorePropagatesExecError(t *testing.T) {
	lanSpeedLimitWriteTestMu.Lock()
	defer lanSpeedLimitWriteTestMu.Unlock()

	originalExec := lanSpeedLimitWriteExec
	t.Cleanup(func() {
		lanSpeedLimitWriteExec = originalExec
	})

	lanSpeedLimitWriteExec = func(ctx context.Context, commands []string) error {
		_ = ctx
		_ = commands
		return errors.New("exec failed")
	}

	store := NewDefaultLanSpeedLimitWriteStore()
	err := store.ApplyPlan(context.Background(), SpeedLimitWritePlan{})
	if err == nil || err.Error() != "exec failed" {
		t.Fatalf("err = %v, want exec failed", err)
	}
}
