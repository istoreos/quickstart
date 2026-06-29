package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

var lanFloatGatewayWriteFacadeTestMu sync.Mutex

type fakeLanFloatGatewayWriteStore struct {
	state        FloatGatewayStateSnapshot
	tags         []FloatGatewayDhcpTagSnapshot
	hosts        []FloatGatewayDhcpHostSnapshot
	readStateErr error
	applyPlan    FloatGatewayWriteExecutionPlan
	applyPlanErr error
}

func (store *fakeLanFloatGatewayWriteStore) ReadState(ctx context.Context) (FloatGatewayStateSnapshot, []FloatGatewayDhcpTagSnapshot, []FloatGatewayDhcpHostSnapshot, error) {
	_ = ctx
	if store.readStateErr != nil {
		return FloatGatewayStateSnapshot{}, nil, nil, store.readStateErr
	}
	return store.state, store.tags, store.hosts, nil
}

func (store *fakeLanFloatGatewayWriteStore) ApplyPlan(ctx context.Context, plan FloatGatewayWriteExecutionPlan) error {
	_ = ctx
	store.applyPlan = plan
	return store.applyPlanErr
}

type fakeLanFloatGatewayApply struct {
	configs []string
	err     error
}

func (apply *fakeLanFloatGatewayApply) Apply(ctx context.Context, configs []string) error {
	_ = ctx
	apply.configs = append([]string{}, configs...)
	return apply.err
}

type fakeLanFloatGatewayWriteFacade struct {
	input FloatGatewayWriteInput
	err   error
}

func (svc *fakeLanFloatGatewayWriteFacade) SetFloatGateway(ctx context.Context, input FloatGatewayWriteInput) error {
	_ = ctx
	svc.input = input
	return svc.err
}

func TestLanFloatGatewayWriteServiceFallbackRoleWritesExpectedState(t *testing.T) {
	t.Parallel()

	store := &fakeLanFloatGatewayWriteStore{}
	apply := &fakeLanFloatGatewayApply{}
	svc := NewLanFloatGatewayWriteService(store, apply)

	err := svc.SetFloatGateway(context.Background(), FloatGatewayWriteInput{
		Enabled: true,
		Role:    "fallback",
		CheckIP: "192.168.100.2",
		SetIP:   "192.168.100.3",
	})
	if err != nil {
		t.Fatalf("SetFloatGateway returned error: %v", err)
	}
	if len(store.applyPlan.FloatCommands) == 0 {
		t.Fatal("expected float commands")
	}
	if len(store.applyPlan.CleanupPlan.DeleteTagSections) != 0 || len(store.applyPlan.CleanupPlan.DeleteHostSections) != 0 {
		t.Fatalf("unexpected cleanup plan: %+v", store.applyPlan.CleanupPlan)
	}
	if len(apply.configs) != 3 || apply.configs[0] != "floatip" || apply.configs[1] != "dhcp" || apply.configs[2] != "dnsmasq" {
		t.Fatalf("configs = %+v", apply.configs)
	}
}

func TestLanFloatGatewayWriteServiceMainRoleWritesExpectedState(t *testing.T) {
	t.Parallel()

	store := &fakeLanFloatGatewayWriteStore{}
	apply := &fakeLanFloatGatewayApply{}
	svc := NewLanFloatGatewayWriteService(store, apply)

	err := svc.SetFloatGateway(context.Background(), FloatGatewayWriteInput{
		Enabled:         true,
		Role:            "main",
		CheckIP:         "192.168.100.2",
		SetIP:           "192.168.100.3",
		CheckURL:        "https://example.com",
		CheckURLTimeout: 8,
	})
	if err != nil {
		t.Fatalf("SetFloatGateway returned error: %v", err)
	}
	last := store.applyPlan.FloatCommands[len(store.applyPlan.FloatCommands)-1]
	if last != "uci add_list floatip.main.check_ip='192.168.100.2'" {
		t.Fatalf("unexpected last command: %q", last)
	}
}

func TestLanFloatGatewayWriteServiceDisablingEnabledGatewayTriggersCleanup(t *testing.T) {
	t.Parallel()

	store := &fakeLanFloatGatewayWriteStore{
		state: FloatGatewayStateSnapshot{Enabled: true, SetIP: "192.168.100.3", CheckIP: "192.168.100.2"},
		tags: []FloatGatewayDhcpTagSnapshot{
			{SectionName: "t_auto_lan1"},
		},
		hosts: []FloatGatewayDhcpHostSnapshot{
			{SectionName: "cfg01", Tag: "t_auto_lan1"},
		},
	}
	apply := &fakeLanFloatGatewayApply{}
	svc := NewLanFloatGatewayWriteService(store, apply)

	err := svc.SetFloatGateway(context.Background(), FloatGatewayWriteInput{
		Enabled: false,
		Role:    "fallback",
		CheckIP: "192.168.100.2",
		SetIP:   "192.168.100.3",
	})
	if err != nil {
		t.Fatalf("SetFloatGateway returned error: %v", err)
	}
	if len(store.applyPlan.CleanupPlan.DeleteTagSections) != 1 || store.applyPlan.CleanupPlan.DeleteTagSections[0] != "t_auto_lan1" {
		t.Fatalf("cleanup tags = %+v", store.applyPlan.CleanupPlan.DeleteTagSections)
	}
	if len(store.applyPlan.CleanupPlan.DeleteHostSections) != 1 || store.applyPlan.CleanupPlan.DeleteHostSections[0] != "cfg01" {
		t.Fatalf("cleanup hosts = %+v", store.applyPlan.CleanupPlan.DeleteHostSections)
	}
}

func TestLanFloatGatewayWriteServiceChangingSetIPOrCheckIPTriggersCleanup(t *testing.T) {
	t.Parallel()

	store := &fakeLanFloatGatewayWriteStore{
		state: FloatGatewayStateSnapshot{Enabled: true, SetIP: "192.168.100.3", CheckIP: "192.168.100.2"},
		tags: []FloatGatewayDhcpTagSnapshot{
			{SectionName: "t_auto_lan1"},
		},
		hosts: []FloatGatewayDhcpHostSnapshot{
			{SectionName: "cfg01", Tag: "t_auto_lan1"},
		},
	}
	apply := &fakeLanFloatGatewayApply{}
	svc := NewLanFloatGatewayWriteService(store, apply)

	err := svc.SetFloatGateway(context.Background(), FloatGatewayWriteInput{
		Enabled: true,
		Role:    "fallback",
		CheckIP: "192.168.100.2",
		SetIP:   "192.168.100.9",
	})
	if err != nil {
		t.Fatalf("SetFloatGateway returned error: %v", err)
	}
	if len(store.applyPlan.CleanupPlan.DeleteTagSections) == 0 || len(store.applyPlan.CleanupPlan.DeleteHostSections) == 0 {
		t.Fatalf("expected cleanup plan, got %+v", store.applyPlan.CleanupPlan)
	}
}

func TestLanFloatGatewayWriteServicePropagatesStoreErrors(t *testing.T) {
	t.Parallel()

	svc := NewLanFloatGatewayWriteService(
		&fakeLanFloatGatewayWriteStore{readStateErr: errors.New("read failed")},
		&fakeLanFloatGatewayApply{},
	)
	err := svc.SetFloatGateway(context.Background(), FloatGatewayWriteInput{Role: "fallback"})
	if err == nil || err.Error() != "read failed" {
		t.Fatalf("err = %v, want read failed", err)
	}
}

func TestLanFloatGatewayWriteServicePropagatesApplyErrors(t *testing.T) {
	t.Parallel()

	store := &fakeLanFloatGatewayWriteStore{}
	apply := &fakeLanFloatGatewayApply{err: errors.New("apply failed")}
	svc := NewLanFloatGatewayWriteService(store, apply)

	err := svc.SetFloatGateway(context.Background(), FloatGatewayWriteInput{
		Enabled: true,
		Role:    "fallback",
		CheckIP: "192.168.100.2",
		SetIP:   "192.168.100.3",
	})
	if err == nil || err.Error() != "apply failed" {
		t.Fatalf("err = %v, want apply failed", err)
	}
}

func TestServiceBackendLanFloatGatewayWriteCompatibilityEnableFloatGatewayDelegatesFallbackFields(t *testing.T) {
	lanFloatGatewayWriteFacadeTestMu.Lock()
	defer lanFloatGatewayWriteFacadeTestMu.Unlock()

	original := newLanFloatGatewayWriteService
	t.Cleanup(func() {
		newLanFloatGatewayWriteService = original
	})

	fake := &fakeLanFloatGatewayWriteFacade{}
	newLanFloatGatewayWriteService = func() lanFloatGatewayWriteFacade {
		return fake
	}

	req := httptest.NewRequest("POST", "/cgi-bin/luci/istore/lanctrl/enableFloatGateway/", strings.NewReader(`{"enabled":true,"role":"fallback","checkIP":"192.168.100.2","setIP":"192.168.100.3"}`))
	resp, err := (&ServiceBackend{}).PostLanEnableFloatGateway(context.Background(), req)
	if err != nil || resp == nil {
		t.Fatalf("PostLanEnableFloatGateway returned resp=%#v err=%v", resp, err)
	}
	if !fake.input.Enabled || fake.input.Role != "fallback" || fake.input.CheckIP != "192.168.100.2" || fake.input.SetIP != "192.168.100.3" {
		t.Fatalf("input = %+v", fake.input)
	}
}

func TestServiceBackendLanFloatGatewayWriteCompatibilityEnableFloatGatewayDelegatesMainFields(t *testing.T) {
	lanFloatGatewayWriteFacadeTestMu.Lock()
	defer lanFloatGatewayWriteFacadeTestMu.Unlock()

	original := newLanFloatGatewayWriteService
	t.Cleanup(func() {
		newLanFloatGatewayWriteService = original
	})

	fake := &fakeLanFloatGatewayWriteFacade{}
	newLanFloatGatewayWriteService = func() lanFloatGatewayWriteFacade {
		return fake
	}

	req := httptest.NewRequest("POST", "/cgi-bin/luci/istore/lanctrl/enableFloatGateway/", strings.NewReader(`{"enabled":true,"role":"main","checkIP":"192.168.100.2","setIP":"192.168.100.3","checkUrl":"https://example.com","checkUrlTimeout":8}`))
	resp, err := (&ServiceBackend{}).PostLanEnableFloatGateway(context.Background(), req)
	if err != nil || resp == nil {
		t.Fatalf("PostLanEnableFloatGateway returned resp=%#v err=%v", resp, err)
	}
	if fake.input.Role != "main" || fake.input.CheckURL != "https://example.com" || fake.input.CheckURLTimeout != 8 {
		t.Fatalf("input = %+v", fake.input)
	}
}
