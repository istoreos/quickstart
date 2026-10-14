package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/digineo/go-uci"
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

	originalApply := lanSpeedLimitWriteApplyPlan
	t.Cleanup(func() {
		lanSpeedLimitWriteApplyPlan = originalApply
	})

	var got SpeedLimitWritePlan
	lanSpeedLimitWriteApplyPlan = func(ctx context.Context, plan SpeedLimitWritePlan) error {
		_ = ctx
		got = plan
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
	if len(got.DeleteSections) != 1 || got.DeleteSections[0].SectionName != "cfg01" || !got.AddSpeedLimit {
		t.Fatalf("unexpected plan: %#v", got)
	}
}

func TestDefaultLanSpeedLimitWriteStoreExecutesDeleteOldBlockBeforeAdd(t *testing.T) {
	lanSpeedLimitWriteTestMu.Lock()
	defer lanSpeedLimitWriteTestMu.Unlock()

	originalApply := lanSpeedLimitWriteApplyPlan
	t.Cleanup(func() {
		lanSpeedLimitWriteApplyPlan = originalApply
	})

	var got SpeedLimitWritePlan
	lanSpeedLimitWriteApplyPlan = func(ctx context.Context, plan SpeedLimitWritePlan) error {
		_ = ctx
		got = plan
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
	if len(got.DeleteSections) != 1 || got.DeleteSections[0].SectionName != "cfg11" || !got.AddBlockRule {
		t.Fatalf("unexpected plan: %#v", got)
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

func TestDefaultLanSpeedLimitWriteStorePropagatesTypedWriteError(t *testing.T) {
	lanSpeedLimitWriteTestMu.Lock()
	defer lanSpeedLimitWriteTestMu.Unlock()

	originalApply := lanSpeedLimitWriteApplyPlan
	t.Cleanup(func() {
		lanSpeedLimitWriteApplyPlan = originalApply
	})

	lanSpeedLimitWriteApplyPlan = func(ctx context.Context, plan SpeedLimitWritePlan) error {
		_ = ctx
		_ = plan
		return errors.New("typed write failed")
	}

	store := NewDefaultLanSpeedLimitWriteStore()
	err := store.ApplyPlan(context.Background(), SpeedLimitWritePlan{})
	if err == nil || err.Error() != "typed write failed" {
		t.Fatalf("err = %v, want typed write failed", err)
	}
}

func TestApplySpeedLimitPlanUsesTypedUCIForUntrustedComment(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "eqos"), []byte("config eqos 'main'\n\toption enabled '1'\n\nconfig device 'legacy'\n\toption ip '192.168.100.20'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(directory, "not-executed")
	comment := "孩子的'电脑; $(touch " + marker + ")"
	if err := applySpeedLimitPlanAt(directory, SpeedLimitWritePlan{Input: SpeedLimitWriteInput{IP: "192.168.100.20", UploadSpeed: 100, DownloadSpeed: 200, Comment: comment}, DeleteSections: []SpeedLimitRuleMatch{{Config: "eqos", SectionName: "legacy"}}, AddSpeedLimit: true}); err != nil {
		t.Fatal(err)
	}
	tree := uci.NewTree(directory)
	if err := tree.LoadConfig("eqos", true); err != nil {
		t.Fatal(err)
	}
	sections, _ := tree.GetSections("eqos", "device")
	if len(sections) != 1 {
		t.Fatalf("device sections=%v", sections)
	}
	stored, _ := tree.GetLast("eqos", sections[0], "comment")
	if stored != safeSpeedPolicyComment(comment) || strings.Contains(stored, "'") {
		t.Fatalf("comment was not safely encoded: %q", stored)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("untrusted comment executed a command")
	}
}
