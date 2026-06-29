package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

var lanSpeedLimitWriteFacadeTestMu sync.Mutex

type fakeLanSpeedLimitWriteStore struct {
	eqosMatches     []SpeedLimitRuleMatch
	firewallMatches []SpeedLimitRuleMatch
	readErr         error
	applyPlan       SpeedLimitWritePlan
	applyPlanErr    error
	moduleInput     SpeedLimitModuleInput
	applyModuleErr  error
}

func (store *fakeLanSpeedLimitWriteStore) ReadRuleMatches(ctx context.Context) ([]SpeedLimitRuleMatch, []SpeedLimitRuleMatch, error) {
	_ = ctx
	if store.readErr != nil {
		return nil, nil, store.readErr
	}
	return store.eqosMatches, store.firewallMatches, nil
}

func (store *fakeLanSpeedLimitWriteStore) ApplyPlan(ctx context.Context, plan SpeedLimitWritePlan) error {
	_ = ctx
	store.applyPlan = plan
	return store.applyPlanErr
}

func (store *fakeLanSpeedLimitWriteStore) ApplyModuleConfig(ctx context.Context, input SpeedLimitModuleInput) error {
	_ = ctx
	store.moduleInput = input
	return store.applyModuleErr
}

type fakeLanSpeedLimitApply struct {
	configs []string
	err     error
}

func (apply *fakeLanSpeedLimitApply) Apply(ctx context.Context, configs []string) error {
	_ = ctx
	apply.configs = append([]string{}, configs...)
	return apply.err
}

func TestLanSpeedLimitWriteServiceUpsertSpeedLimitRulePrefersEqosAndRemovesOldBlock(t *testing.T) {
	t.Parallel()

	store := &fakeLanSpeedLimitWriteStore{
		firewallMatches: []SpeedLimitRuleMatch{{Config: "firewall", SectionName: "cfg11", MatchMAC: "AA:BB:CC:DD:EE:20"}},
	}
	apply := &fakeLanSpeedLimitApply{}
	svc := NewLanSpeedLimitWriteService(store, apply)

	err := svc.UpsertSpeedLimitRule(context.Background(), SpeedLimitWriteInput{
		Action:        "add",
		IP:            "192.168.100.20",
		MAC:           "aa:bb:cc:dd:ee:20",
		NetworkAccess: true,
		UploadSpeed:   300,
		DownloadSpeed: 2000,
		Comment:       "tablet",
	})
	if err != nil {
		t.Fatalf("UpsertSpeedLimitRule returned error: %v", err)
	}
	if !store.applyPlan.AddSpeedLimit || store.applyPlan.AddBlockRule {
		t.Fatalf("applyPlan = %+v", store.applyPlan)
	}
	if len(store.applyPlan.DeleteSections) != 1 || store.applyPlan.DeleteSections[0].SectionName != "cfg11" {
		t.Fatalf("DeleteSections = %+v", store.applyPlan.DeleteSections)
	}
	if len(apply.configs) != 2 || apply.configs[0] != "eqos" || apply.configs[1] != "firewall" {
		t.Fatalf("configs = %+v", apply.configs)
	}
}

func TestLanSpeedLimitWriteServiceUpsertBlockRuleRemovesOldEqos(t *testing.T) {
	t.Parallel()

	store := &fakeLanSpeedLimitWriteStore{
		eqosMatches: []SpeedLimitRuleMatch{{Config: "eqos", SectionName: "cfg01", MatchIP: "192.168.100.20"}},
	}
	apply := &fakeLanSpeedLimitApply{}
	svc := NewLanSpeedLimitWriteService(store, apply)

	err := svc.UpsertSpeedLimitRule(context.Background(), SpeedLimitWriteInput{
		Action:        "add",
		IP:            "192.168.100.20",
		MAC:           "AA:BB:CC:DD:EE:20",
		NetworkAccess: false,
	})
	if err != nil {
		t.Fatalf("UpsertSpeedLimitRule returned error: %v", err)
	}
	if store.applyPlan.AddSpeedLimit || !store.applyPlan.AddBlockRule {
		t.Fatalf("applyPlan = %+v", store.applyPlan)
	}
	if len(store.applyPlan.DeleteSections) != 1 || store.applyPlan.DeleteSections[0].SectionName != "cfg01" {
		t.Fatalf("DeleteSections = %+v", store.applyPlan.DeleteSections)
	}
}

func TestLanSpeedLimitWriteServicePropagatesStoreErrors(t *testing.T) {
	t.Parallel()

	svc := NewLanSpeedLimitWriteService(
		&fakeLanSpeedLimitWriteStore{readErr: errors.New("read failed")},
		&fakeLanSpeedLimitApply{},
	)
	err := svc.UpsertSpeedLimitRule(context.Background(), SpeedLimitWriteInput{
		Action:        "add",
		MAC:           "AA:BB:CC:DD:EE:20",
		NetworkAccess: false,
	})
	if err == nil || err.Error() != "read failed" {
		t.Fatalf("err = %v, want read failed", err)
	}
}

func TestLanSpeedLimitWriteServicePropagatesApplyErrors(t *testing.T) {
	t.Parallel()

	store := &fakeLanSpeedLimitWriteStore{
		firewallMatches: []SpeedLimitRuleMatch{{Config: "firewall", SectionName: "cfg11", MatchMAC: "AA:BB:CC:DD:EE:20"}},
	}
	apply := &fakeLanSpeedLimitApply{err: errors.New("apply failed")}
	svc := NewLanSpeedLimitWriteService(store, apply)

	err := svc.UpsertSpeedLimitRule(context.Background(), SpeedLimitWriteInput{
		Action:        "delete",
		MAC:           "AA:BB:CC:DD:EE:20",
		NetworkAccess: false,
	})
	if err == nil || err.Error() != "apply failed" {
		t.Fatalf("err = %v, want apply failed", err)
	}
}

func TestLanSpeedLimitWriteServiceSetSpeedLimitModuleAppliesEqosOnly(t *testing.T) {
	t.Parallel()

	store := &fakeLanSpeedLimitWriteStore{}
	apply := &fakeLanSpeedLimitApply{}
	svc := NewLanSpeedLimitWriteService(store, apply)

	err := svc.SetSpeedLimitModule(context.Background(), SpeedLimitModuleInput{
		Enabled:       true,
		UploadSpeed:   0,
		DownloadSpeed: 0,
	})
	if err != nil {
		t.Fatalf("SetSpeedLimitModule returned error: %v", err)
	}
	if store.moduleInput.UploadSpeed != 200 || store.moduleInput.DownloadSpeed != 2000 {
		t.Fatalf("moduleInput = %+v", store.moduleInput)
	}
	if len(apply.configs) != 1 || apply.configs[0] != "eqos" {
		t.Fatalf("configs = %+v", apply.configs)
	}
}

type fakeLanSpeedLimitWriteFacade struct {
	upsertInput SpeedLimitWriteInput
	upsertErr   error
	moduleInput SpeedLimitModuleInput
	moduleErr   error
}

func (svc *fakeLanSpeedLimitWriteFacade) UpsertSpeedLimitRule(ctx context.Context, input SpeedLimitWriteInput) error {
	_ = ctx
	svc.upsertInput = input
	return svc.upsertErr
}

func (svc *fakeLanSpeedLimitWriteFacade) SetSpeedLimitModule(ctx context.Context, input SpeedLimitModuleInput) error {
	_ = ctx
	svc.moduleInput = input
	return svc.moduleErr
}

func TestServiceBackendLanSpeedLimitWriteCompatibilitySpeedLimitConfigDelegatesToService(t *testing.T) {
	lanSpeedLimitWriteFacadeTestMu.Lock()
	defer lanSpeedLimitWriteFacadeTestMu.Unlock()

	original := newLanSpeedLimitWriteService
	t.Cleanup(func() {
		newLanSpeedLimitWriteService = original
	})

	fake := &fakeLanSpeedLimitWriteFacade{}
	newLanSpeedLimitWriteService = func() lanSpeedLimitWriteFacade {
		return fake
	}

	req := httptest.NewRequest("POST", "/cgi-bin/luci/istore/lanctrl/speedLimitConfig/", strings.NewReader(`{"action":"add","ip":"192.168.100.20","mac":"aa:bb:cc:dd:ee:20","downloadSpeed":2000,"uploadSpeed":300,"comment":"tablet","networkAccess":false}`))
	resp, err := (&ServiceBackend{}).PostLanSpeedLimitConfig(context.Background(), req)
	if err != nil || resp == nil {
		t.Fatalf("PostLanSpeedLimitConfig returned resp=%#v err=%v", resp, err)
	}
	if fake.upsertInput.MAC != "AA:BB:CC:DD:EE:20" {
		t.Fatalf("MAC = %q, want uppercased value", fake.upsertInput.MAC)
	}
	if fake.upsertInput.Action != "add" || fake.upsertInput.IP != "192.168.100.20" || fake.upsertInput.Comment != "tablet" {
		t.Fatalf("upsertInput = %+v", fake.upsertInput)
	}
	if fake.upsertInput.NetworkAccess {
		t.Fatalf("expected NetworkAccess=false, got %+v", fake.upsertInput)
	}
}

func TestServiceBackendLanSpeedLimitWriteCompatibilityEnableSpeedLimitDelegatesDefaults(t *testing.T) {
	lanSpeedLimitWriteFacadeTestMu.Lock()
	defer lanSpeedLimitWriteFacadeTestMu.Unlock()

	original := newLanSpeedLimitWriteService
	t.Cleanup(func() {
		newLanSpeedLimitWriteService = original
	})

	fake := &fakeLanSpeedLimitWriteFacade{}
	newLanSpeedLimitWriteService = func() lanSpeedLimitWriteFacade {
		return fake
	}

	req := httptest.NewRequest("POST", "/cgi-bin/luci/istore/lanctrl/enableSpeedLimit/", strings.NewReader(`{"enabled":true}`))
	resp, err := (&ServiceBackend{}).PostLanEnableSpeedLimit(context.Background(), req)
	if err != nil || resp == nil {
		t.Fatalf("PostLanEnableSpeedLimit returned resp=%#v err=%v", resp, err)
	}
	if !fake.moduleInput.Enabled || fake.moduleInput.UploadSpeed != 200 || fake.moduleInput.DownloadSpeed != 2000 {
		t.Fatalf("moduleInput = %+v", fake.moduleInput)
	}
}
