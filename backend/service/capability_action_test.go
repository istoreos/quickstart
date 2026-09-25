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
	installs         []string
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
	if err != nil || !preview.Result.Plan.CanApply || preview.Result.Plan.DraftToken != request.DraftToken || preview.Result.Plan.Target != "app-meta-eqos" {
		t.Fatalf("preview = %#v, %v", preview, err)
	}
	cancelled, err := module.Apply(context.Background(), request)
	if err != nil || !cancelled.Result.Cancelled || cancelled.Result.Changed || len(store.installs) != 0 || cancelled.Result.Plan.DraftToken != request.DraftToken {
		t.Fatalf("cancel = %#v, %v", cancelled, err)
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
