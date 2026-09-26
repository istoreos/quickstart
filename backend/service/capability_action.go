package service

import (
	"context"
	"errors"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/istoreos/quickstart/backend/models"
	"golang.org/x/sys/unix"
)

const capabilityInstallRequiredFreeBytes int64 = 8 << 20

var capabilityDraftTokenPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

type capabilityActionStore interface {
	Read(context.Context, string) (*models.Capability, error)
	AvailableBytes(context.Context) (int64, error)
	Install(context.Context, string) error
	Enable(context.Context, string) error
}

type CapabilityActionModule struct {
	mu           sync.Mutex
	store        capabilityActionStore
	pollInterval time.Duration
	pollTimeout  time.Duration
}

func NewCapabilityActionModule(store capabilityActionStore) *CapabilityActionModule {
	return &CapabilityActionModule{store: store, pollInterval: 500 * time.Millisecond, pollTimeout: 30 * time.Second}
}
func NewDefaultCapabilityActionModule() *CapabilityActionModule {
	return NewCapabilityActionModule(&systemCapabilityActionStore{})
}

func (module *CapabilityActionModule) Plan(ctx context.Context, request *models.CapabilityActionRequest) (*models.CapabilityActionResponse, error) {
	plan, err := module.plan(ctx, request)
	if err != nil {
		return nil, err
	}
	return capabilityActionResponse(plan, false, false, plan.Current, nil), nil
}

func (module *CapabilityActionModule) Apply(ctx context.Context, request *models.CapabilityActionRequest) (*models.CapabilityActionResponse, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	plan, err := module.plan(ctx, request)
	if err != nil {
		return nil, err
	}
	if request == nil || !request.Confirm {
		return capabilityActionResponse(plan, false, true, plan.Current, nil), nil
	}
	if request.ExpectedState != "" && models.CapabilityState(request.ExpectedState) != plan.Current.State {
		return capabilityActionResponse(plan, false, false, plan.Current, &models.DevicePolicyError{Code: "conflict", Message: "capability state changed; review the action again"}), nil
	}
	if !plan.CanApply {
		return capabilityActionResponse(plan, false, false, plan.Current, plan.Error), nil
	}
	pendingState, pendingCode := models.CapabilityState("not_installed"), "install_pending"
	if plan.Action == "enable" {
		pendingState, pendingCode = "disabled", "enable_pending"
		if err := module.store.Enable(ctx, plan.CapabilityKey); err != nil {
			return capabilityActionResponse(plan, false, false, plan.Current, &models.DevicePolicyError{Code: "enable_failed", Message: err.Error()}), nil
		}
	} else if err := module.store.Install(ctx, plan.Target); err != nil {
		return capabilityActionResponse(plan, false, false, plan.Current, &models.DevicePolicyError{Code: "install_failed", Message: err.Error()}), nil
	}
	current, pending, err := module.waitForStateChange(ctx, plan.CapabilityKey, pendingState)
	if err != nil {
		return capabilityActionResponse(plan, false, false, nil, &models.DevicePolicyError{Code: "status_unavailable", Message: err.Error()}), nil
	}
	if pending {
		return capabilityActionResponse(plan, false, false, current, &models.DevicePolicyError{Code: pendingCode, Message: "component state is still changing; check status again shortly"}), nil
	}
	return capabilityActionResponse(plan, true, false, current, nil), nil
}

func (module *CapabilityActionModule) waitForStateChange(ctx context.Context, capabilityKey string, pendingState models.CapabilityState) (*models.Capability, bool, error) {
	interval, timeout := module.pollInterval, module.pollTimeout
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		current, err := module.store.Read(ctx, capabilityKey)
		if err != nil {
			return nil, false, err
		}
		if current.State != pendingState {
			return current, false, nil
		}
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		case <-deadline.C:
			return current, true, nil
		case <-ticker.C:
		}
	}
}

func (module *CapabilityActionModule) plan(ctx context.Context, request *models.CapabilityActionRequest) (*models.CapabilityActionPlan, error) {
	if module == nil || module.store == nil {
		return nil, errors.New("capability action module is unavailable")
	}
	plan := &models.CapabilityActionPlan{RequiresConfirmation: true}
	if request == nil || (request.Action != "install" && request.Action != "enable") || (request.DraftToken != "" && !capabilityDraftTokenPattern.MatchString(request.DraftToken)) {
		plan.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "invalid capability action request"}
		return plan, nil
	}
	plan.CapabilityKey, plan.Action, plan.DraftToken = request.CapabilityKey, request.Action, request.DraftToken
	if request.Action == "install" {
		plan.Target = capabilityInstallTarget(request.CapabilityKey)
		plan.RequiredFreeBytes = capabilityInstallRequiredFreeBytes
	} else if request.CapabilityKey == "device_speed_limit" {
		plan.Target = request.CapabilityKey
	}
	if plan.Target == "" {
		plan.Error = &models.DevicePolicyError{Code: "unsupported", Message: "this capability action is not supported on this system"}
		return plan, nil
	}
	current, err := module.store.Read(ctx, request.CapabilityKey)
	if err != nil {
		plan.Error = &models.DevicePolicyError{Code: "status_unavailable", Message: err.Error()}
		return plan, nil
	}
	plan.Current = current
	if request.Action == "enable" {
		if current.State != "disabled" {
			plan.Error = &models.DevicePolicyError{Code: "conflict", Message: "capability is no longer waiting to be enabled"}
		} else {
			plan.CanApply = true
		}
		return plan, nil
	}
	available, err := module.store.AvailableBytes(ctx)
	if err != nil {
		plan.Error = &models.DevicePolicyError{Code: "status_unavailable", Message: "available storage could not be checked"}
		return plan, nil
	}
	plan.AvailableFreeBytes = available
	if current.State != "not_installed" {
		plan.Error = &models.DevicePolicyError{Code: "conflict", Message: "capability is no longer waiting for installation"}
	} else if available < capabilityInstallRequiredFreeBytes {
		plan.Error = &models.DevicePolicyError{Code: "insufficient_space", Message: "not enough free storage to install this component"}
	} else {
		plan.CanApply = true
	}
	return plan, nil
}

func capabilityInstallTarget(key string) string {
	switch key {
	case "device_speed_limit":
		return "app-meta-eqos"
	case "floating_gateway":
		return "app-meta-floatip"
	default:
		return ""
	}
}

func capabilityActionResponse(plan *models.CapabilityActionPlan, changed, cancelled bool, current *models.Capability, policyErr *models.DevicePolicyError) *models.CapabilityActionResponse {
	return &models.CapabilityActionResponse{Result: &models.CapabilityActionResult{Plan: plan, Changed: changed, Cancelled: cancelled, Current: current, Error: policyErr}}
}

type systemCapabilityActionStore struct{}

func (store *systemCapabilityActionStore) Read(ctx context.Context, key string) (*models.Capability, error) {
	response, err := NewLanGlobalConfigService().GetGlobalConfigs(ctx)
	if err != nil {
		return nil, err
	}
	if response == nil || response.Result == nil || response.Result.Capabilities == nil {
		return nil, errors.New("capability status is unavailable")
	}
	capability := response.Result.Capabilities.Items[key]
	if capability == nil {
		return &models.Capability{State: "unsupported", Reason: "unsupported_capability"}, nil
	}
	copy := *capability
	return &copy, nil
}
func (store *systemCapabilityActionStore) AvailableBytes(context.Context) (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs("/", &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}
func (store *systemCapabilityActionStore) Install(ctx context.Context, target string) error {
	output, err := exec.CommandContext(ctx, "is-opkg", "install", target).CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if len(detail) > 4096 {
			detail = detail[len(detail)-4096:]
		}
		log.Printf("capability installation failed target=%s: %s", target, detail)
		return errors.New("installation failed; check software sources and firmware compatibility")
	}
	return nil
}

func (store *systemCapabilityActionStore) Enable(ctx context.Context, capabilityKey string) error {
	if capabilityKey != "device_speed_limit" {
		return errors.New("this capability cannot be enabled from device management")
	}
	previous, err := exec.CommandContext(ctx, "uci", "get", "eqos.@eqos[0].enabled").Output()
	if err != nil {
		return errors.New("limit service configuration is unavailable")
	}
	oldValue := strings.TrimSpace(string(previous))
	rollback := func() {
		_ = exec.Command("uci", "set", "eqos.@eqos[0].enabled="+oldValue).Run()
		_ = exec.Command("uci", "commit", "eqos").Run()
		_ = exec.Command("/etc/init.d/eqos", "restart").Run()
	}
	if err := exec.CommandContext(ctx, "uci", "set", "eqos.@eqos[0].enabled=1").Run(); err != nil {
		return errors.New("limit service could not be enabled")
	}
	if err := exec.CommandContext(ctx, "uci", "commit", "eqos").Run(); err != nil {
		rollback()
		return errors.New("limit service configuration could not be saved")
	}
	if output, err := exec.CommandContext(ctx, "/etc/init.d/eqos", "restart").CombinedOutput(); err != nil {
		log.Printf("capability enable failed key=%s: %s", capabilityKey, strings.TrimSpace(string(output)))
		rollback()
		return errors.New("limit service could not be restarted; original setting was restored")
	}
	return nil
}
