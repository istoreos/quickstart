package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

type fakeCapabilityActionStore struct {
	mu               sync.Mutex
	capability       *models.Capability
	available        int64
	readErr          error
	spaceErr         error
	installErr       error
	enableErr        error
	installs         []string
	enables          []string
	installDelay     time.Duration
	stayNotInstalled bool
}

func (store *fakeCapabilityActionStore) Read(context.Context, string) (*models.Capability, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.readErr != nil {
		return nil, store.readErr
	}
	copy := *store.capability
	return &copy, nil
}
func (store *fakeCapabilityActionStore) AvailableBytes(context.Context) (int64, error) {
	return store.available, store.spaceErr
}
func (store *fakeCapabilityActionStore) Install(_ context.Context, target string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.installs = append(store.installs, target)
	if store.installErr == nil && !store.stayNotInstalled && store.installDelay == 0 {
		store.capability = &models.Capability{State: "disabled", Actions: []*models.CapabilityAction{{Kind: "enable", RequiresConfirmation: true}}}
	}
	if store.installErr == nil && !store.stayNotInstalled && store.installDelay > 0 {
		delay := store.installDelay
		go func() {
			time.Sleep(delay)
			store.mu.Lock()
			store.capability = &models.Capability{State: "disabled", Actions: []*models.CapabilityAction{{Kind: "enable", RequiresConfirmation: true}}}
			store.mu.Unlock()
		}()
	}
	return store.installErr
}

func (store *fakeCapabilityActionStore) Enable(_ context.Context, capabilityKey string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.enables = append(store.enables, capabilityKey)
	if store.enableErr == nil {
		store.capability = &models.Capability{State: "available"}
	}
	return store.enableErr
}

func TestCapabilityEnableRequiresPreviewAndEnablesInstalledService(t *testing.T) {
	store := &fakeCapabilityActionStore{capability: &models.Capability{State: "disabled"}}
	module := NewCapabilityActionModule(store)
	request := &models.CapabilityActionRequest{CapabilityKey: "device_speed_limit", Action: "enable", DraftToken: "device-1:restrictions"}
	preview, err := module.Plan(context.Background(), request)
	if err != nil || preview.Result.Plan.Error != nil || !preview.Result.Plan.CanApply || preview.Result.Plan.RequiredFreeBytes != 0 {
		t.Fatalf("enable preview = %#v, %v", preview, err)
	}
	cancelled, err := module.Apply(context.Background(), request)
	if err != nil || !cancelled.Result.Cancelled || len(store.enables) != 0 {
		t.Fatalf("enable cancel = %#v, %v", cancelled, err)
	}
	request.Confirm, request.ExpectedState = true, "disabled"
	applied, err := module.Apply(context.Background(), request)
	if err != nil || applied.Result.Error != nil || !applied.Result.Changed || applied.Result.Current.State != "available" || len(store.enables) != 1 {
		t.Fatalf("enable apply = %#v, %v", applied, err)
	}

	store.capability = &models.Capability{State: "available"}
	conflict, err := module.Plan(context.Background(), request)
	if err != nil || conflict.Result.Plan.Error == nil || conflict.Result.Plan.Error.Code != "conflict" {
		t.Fatalf("enable conflict = %#v, %v", conflict, err)
	}
}

func TestCapabilityInstallWaitsForAsyncInstallerAndReportsBoundedPending(t *testing.T) {
	store := &fakeCapabilityActionStore{capability: &models.Capability{State: "not_installed"}, available: capabilityInstallRequiredFreeBytes, installDelay: 5 * time.Millisecond}
	module := NewCapabilityActionModule(store)
	module.pollInterval, module.pollTimeout = time.Millisecond, 50*time.Millisecond
	response, err := module.Apply(context.Background(), &models.CapabilityActionRequest{CapabilityKey: "device_speed_limit", Action: "install", Confirm: true})
	if err != nil || response.Result.Error != nil || !response.Result.Changed || response.Result.Current.State != "disabled" {
		t.Fatalf("async install = %#v, %v", response, err)
	}

	store = &fakeCapabilityActionStore{capability: &models.Capability{State: "not_installed"}, available: capabilityInstallRequiredFreeBytes, stayNotInstalled: true}
	module = NewCapabilityActionModule(store)
	module.pollInterval, module.pollTimeout = time.Millisecond, 5*time.Millisecond
	response, err = module.Apply(context.Background(), &models.CapabilityActionRequest{CapabilityKey: "device_speed_limit", Action: "install", Confirm: true})
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "install_pending" || response.Result.Changed {
		t.Fatalf("pending install = %#v, %v", response, err)
	}
}

func TestCapabilityInstallPlanAndCancelPreserveDraft(t *testing.T) {
	store := &fakeCapabilityActionStore{capability: &models.Capability{State: "not_installed"}, available: capabilityInstallRequiredFreeBytes}
	module := NewCapabilityActionModule(store)
	request := &models.CapabilityActionRequest{CapabilityKey: "device_speed_limit", Action: "install", DraftToken: "device-1:restrictions"}
	preview, err := module.Plan(context.Background(), request)
	if err != nil || !preview.Result.Plan.CanApply || preview.Result.Plan.DraftToken != request.DraftToken || preview.Result.Plan.Target != "quickstart-netpolicy" {
		t.Fatalf("preview = %#v, %v", preview, err)
	}
	cancelled, err := module.Apply(context.Background(), request)
	if err != nil || !cancelled.Result.Cancelled || cancelled.Result.Changed || len(store.installs) != 0 || cancelled.Result.Plan.DraftToken != request.DraftToken {
		t.Fatalf("cancel = %#v, %v", cancelled, err)
	}
}

func TestSystemCapabilityActionStoreEnablesNativeServiceWithoutTouchingEqos(t *testing.T) {
	type invocation struct {
		name string
		args []string
	}
	var calls []invocation
	store := &systemCapabilityActionStore{run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, invocation{name: name, args: append([]string(nil), args...)})
		if len(args) == 1 && (args[0] == "enabled" || args[0] == "running") {
			return nil, errors.New("inactive")
		}
		return nil, nil
	}}
	if err := store.Enable(context.Background(), "device_speed_limit"); err != nil {
		t.Fatal(err)
	}
	want := []string{"enabled", "running", "enable", "start"}
	if len(calls) != len(want) {
		t.Fatalf("calls = %#v", calls)
	}
	for index, call := range calls {
		if call.name != "/etc/init.d/quickstart-netpolicy" || len(call.args) != 1 || call.args[0] != want[index] {
			t.Fatalf("call[%d] = %#v", index, call)
		}
		if call.name == "uci" || call.name == "/etc/init.d/eqos" {
			t.Fatalf("legacy service touched: %#v", call)
		}
	}
}

func TestSystemCapabilityActionStoreRestoresNativeServiceStateWhenStartFails(t *testing.T) {
	var actions []string
	store := &systemCapabilityActionStore{run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "/etc/init.d/quickstart-netpolicy" || len(args) != 1 {
			t.Fatalf("unexpected command: %s %#v", name, args)
		}
		actions = append(actions, args[0])
		switch args[0] {
		case "enabled", "running":
			return nil, errors.New("inactive")
		case "start":
			return []byte("start failed"), errors.New("exit 1")
		default:
			return nil, nil
		}
	}}
	err := store.Enable(context.Background(), "device_speed_limit")
	if err == nil || err.Error() != "limit service could not be started; original service state was restored" {
		t.Fatalf("error = %v", err)
	}
	want := []string{"enabled", "running", "enable", "start", "stop", "disable"}
	if len(actions) != len(want) {
		t.Fatalf("actions = %#v", actions)
	}
	for index := range want {
		if actions[index] != want[index] {
			t.Fatalf("actions = %#v", actions)
		}
	}
}

func TestCapabilityInstallSuccessReturnsWithoutApplyingPolicy(t *testing.T) {
	store := &fakeCapabilityActionStore{capability: &models.Capability{State: "not_installed"}, available: capabilityInstallRequiredFreeBytes}
	module := NewCapabilityActionModule(store)
	response, err := module.Apply(context.Background(), &models.CapabilityActionRequest{CapabilityKey: "floating_gateway", Action: "install", DraftToken: "device-1:network", Confirm: true, ExpectedState: "not_installed"})
	if err != nil || response.Result.Error != nil || !response.Result.Changed || response.Result.Current.State != "disabled" || len(store.installs) != 1 {
		t.Fatalf("install = %#v, %v", response, err)
	}
	if store.installs[0] != "app-meta-floatip" {
		t.Fatalf("target = %q", store.installs[0])
	}
}

func TestCapabilityInstallFailureAndChangedStateAreSafe(t *testing.T) {
	store := &fakeCapabilityActionStore{capability: &models.Capability{State: "not_installed"}, available: capabilityInstallRequiredFreeBytes, installErr: errors.New("feed unavailable")}
	module := NewCapabilityActionModule(store)
	response, err := module.Apply(context.Background(), &models.CapabilityActionRequest{CapabilityKey: "device_speed_limit", Action: "install", Confirm: true})
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "install_failed" {
		t.Fatalf("failure = %#v, %v", response, err)
	}
	store.capability = &models.Capability{State: "available"}
	response, err = module.Apply(context.Background(), &models.CapabilityActionRequest{CapabilityKey: "device_speed_limit", Action: "install", Confirm: true, ExpectedState: "not_installed"})
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "conflict" {
		t.Fatalf("changed state = %#v, %v", response, err)
	}
}

func TestCapabilityInstallRejectsLowSpaceUnsupportedAndBadDraft(t *testing.T) {
	store := &fakeCapabilityActionStore{capability: &models.Capability{State: "not_installed"}, available: capabilityInstallRequiredFreeBytes - 1}
	module := NewCapabilityActionModule(store)
	response, _ := module.Plan(context.Background(), &models.CapabilityActionRequest{CapabilityKey: "device_speed_limit", Action: "install"})
	if response.Result.Plan.Error == nil || response.Result.Plan.Error.Code != "insufficient_space" {
		t.Fatalf("space plan = %#v", response.Result.Plan)
	}
	response, _ = module.Plan(context.Background(), &models.CapabilityActionRequest{CapabilityKey: "traffic_details", Action: "install"})
	if response.Result.Plan.Error == nil || response.Result.Plan.Error.Code != "unsupported" {
		t.Fatalf("unsupported plan = %#v", response.Result.Plan)
	}
	response, _ = module.Plan(context.Background(), &models.CapabilityActionRequest{CapabilityKey: "device_speed_limit", Action: "install", DraftToken: "bad token"})
	if response.Result.Plan.Error == nil || response.Result.Plan.Error.Code != "validation_failed" {
		t.Fatalf("draft plan = %#v", response.Result.Plan)
	}
}
