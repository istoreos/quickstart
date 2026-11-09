package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/digineo/go-uci"
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
	operations   componentOperations
	store        capabilityActionStore
	pollInterval time.Duration
	pollTimeout  time.Duration
}

func NewCapabilityActionModule(store capabilityActionStore) *CapabilityActionModule {
	return &CapabilityActionModule{store: store, pollInterval: 500 * time.Millisecond, pollTimeout: 30 * time.Second}
}
func NewDefaultCapabilityActionModule() *CapabilityActionModule {
	return NewCapabilityActionModule(newSystemCapabilityActionStore())
}

func (module *CapabilityActionModule) Plan(ctx context.Context, request *models.CapabilityActionRequest) (*models.CapabilityActionResponse, error) {
	plan, err := module.plan(ctx, request)
	if err != nil {
		return nil, err
	}
	return capabilityActionResponse(plan, false, false, plan.Current, nil), nil
}

func (module *CapabilityActionModule) Apply(ctx context.Context, request *models.CapabilityActionRequest) (*models.CapabilityActionResponse, error) {
	if !module.mu.TryLock() {
		return capabilityActionResponse(nil, false, false, nil, &models.DevicePolicyError{Code: "component_busy", Message: "another component action is in progress; check status before retrying"}), nil
	}
	defer module.mu.Unlock()
	if request != nil && request.Action == "enable" {
		locked, release, err := lockLANWrite(ctx)
		if err != nil {
			return nil, err
		}
		defer release()
		ctx = locked
	}
	plan, err := module.plan(ctx, request)
	if err != nil {
		return nil, err
	}
	if request == nil || !request.Confirm {
		return capabilityActionResponse(plan, false, true, plan.Current, nil), nil
	}
	if !plan.CanApply || plan.Current == nil {
		return capabilityActionResponse(plan, false, false, plan.Current, plan.Error), nil
	}
	if (request.ExpectedVersion != "" && request.ExpectedVersion != plan.Version) || (request.ExpectedState != "" && models.CapabilityState(request.ExpectedState) != plan.Current.State) {
		return capabilityActionResponse(plan, false, false, plan.Current, &models.DevicePolicyError{Code: "conflict", Message: "capability state changed; review the action again"}), nil
	}
	// Closing a panel or losing HTTP must not kill a package manager halfway
	// through installation. State reads remain independent and no POST is queued.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Minute)
	defer cancel()
	module.operation(plan.CapabilityKey, &componentOperation{state: "running", action: plan.Action})
	defer func() {
		if module.operation(plan.CapabilityKey, nil).state == "running" {
			module.operation(plan.CapabilityKey, &componentOperation{})
		}
	}()
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
		module.operation(plan.CapabilityKey, &componentOperation{state: "pending", action: plan.Action})
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
	if operation := module.operation(request.CapabilityKey, nil); operation.state != "" {
		plan.Error = &models.DevicePolicyError{Code: "component_pending", Message: "component action is still pending; check status before submitting again"}
		return plan, nil
	}
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
	plan.NetworkID = selectedLAN(ctx)
	plan.OperationScope = "network"
	plan.AffectedNetworks = []string{selectedLAN(ctx)}
	if request.Action == "install" {
		plan.OperationScope = "component"
		if resolver, _ := ctx.Value(lanScopeResolverKey{}).(*LANScopeModule); resolver != nil {
			if scopes, err := resolver.List(ctx); err == nil {
				plan.AffectedNetworks = nil
				for _, scope := range scopes {
					plan.AffectedNetworks = append(plan.AffectedNetworks, scope.NetworkID)
				}
			}
		}
	}
	version := plan.CapabilityKey + "\x00" + plan.Action + "\x00" + string(current.State)
	if system, ok := module.store.(*systemCapabilityActionStore); ok {
		version += "\x00" + hashOptionalMigrationFile(filepath.Join(system.nativeConfigDir(), "quickstart-netpolicy"))
	}
	sum := sha256.Sum256([]byte(version))
	plan.Version = bindLANVersion(ctx, hex.EncodeToString(sum[:]))
	if request.Action == "enable" {
		if system, ok := module.store.(*systemCapabilityActionStore); ok {
			plan.AffectedNetworks = system.enabledNetworks(selectedLAN(ctx))
			if len(plan.AffectedNetworks) > 1 {
				plan.OperationScope = "shared_service"
			}
		}
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
		return "quickstart-netpolicy"
	case "floating_gateway":
		return "app-meta-floatip"
	case "compat_speed_limit":
		return "luci-app-eqos"
	default:
		return ""
	}
}

func capabilityActionResponse(plan *models.CapabilityActionPlan, changed, cancelled bool, current *models.Capability, policyErr *models.DevicePolicyError) *models.CapabilityActionResponse {
	return &models.CapabilityActionResponse{Result: &models.CapabilityActionResult{Plan: plan, Changed: changed, Cancelled: cancelled, Current: current, Error: policyErr}}
}

type capabilityCommandRunner func(context.Context, string, ...string) ([]byte, error)

type systemCapabilityActionStore struct {
	configDir   string
	servicePath string
	stat        func(string) (os.FileInfo, error)
	run         capabilityCommandRunner
}

func newSystemCapabilityActionStore() *systemCapabilityActionStore {
	return &systemCapabilityActionStore{run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, name, args...).CombinedOutput()
	}}
}

func (store *systemCapabilityActionStore) runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	if store == nil || store.run == nil {
		return nil, errors.New("capability command runner is unavailable")
	}
	return store.run(ctx, name, args...)
}

func (store *systemCapabilityActionStore) Read(ctx context.Context, key string) (*models.Capability, error) {
	if selectedLAN(ctx) != defaultLAN && key != "device_speed_limit" {
		return &models.Capability{State: "unsupported", Reason: "provider_scope_unsupported"}, nil
	}
	if key == "device_speed_limit" {
		if selectedLANScope(ctx) != nil {
			return store.scopedNativeCapability(ctx)
		}
		return nativeComponentCapability(ctx, nativeProviderCapability, store.runCommand)
	}
	var installed, enabled bool
	var err error
	switch key {
	case "floating_gateway":
		installed, err = componentPackageInstalled("floatip", os.Stat)
		if installed && err == nil {
			tree := uci.NewTree("/etc/config")
			err = tree.LoadConfig("floatip", true)
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
			value, _ := tree.GetLast("floatip", "main", "enabled")
			enabled = value == "1"
		}
	case "compat_speed_limit":
		installed, err = componentPackageInstalled("luci-app-eqos", os.Stat)
		enabled = installed // Selecting/enabling eqos uses the existing settings transaction.
	default:
		return &models.Capability{State: "unsupported", Reason: "unsupported_capability"}, nil
	}
	status := buildDeviceManagementCapability(installed, enabled, err)
	decorateCapabilityActions(status, capabilityInstallTarget(key))
	// Floating gateway enable/configuration is a previewed network task, not a
	// generic service-start action. The UI links back to its existing wizard.
	if status.State == "disabled" {
		status.Actions = nil
	}
	return status, err
}
func (store *systemCapabilityActionStore) AvailableBytes(context.Context) (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs("/", &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}
func (store *systemCapabilityActionStore) Install(ctx context.Context, target string) error {
	if _, err := store.runCommand(ctx, "is-opkg", "update"); err != nil {
		return errors.New("software sources could not be updated; check the software center and retry")
	}
	_, err := store.runCommand(ctx, "is-opkg", "install", target)
	if err != nil {
		// Package-manager output can contain authenticated feed URLs.
		log.Printf("capability installation failed target=%s", target)
		return errors.New("installation failed; check software sources and firmware compatibility")
	}
	return nil
}

func (store *systemCapabilityActionStore) Enable(ctx context.Context, capabilityKey string) error {
	if selectedLANScope(ctx) != nil {
		return store.enableNativeScope(ctx, capabilityKey)
	}
	return store.enableLegacyNative(ctx, capabilityKey)
}

func (store *systemCapabilityActionStore) enableLegacyNative(ctx context.Context, capabilityKey string) error {
	if capabilityKey != "device_speed_limit" {
		return errors.New("this capability cannot be enabled from device management")
	}
	const service = "/etc/init.d/quickstart-netpolicy"
	// procd's boot symlink and the daemon's UCI enabled switch are separate.
	// Starting the init script with main.enabled=0 silently starts no process.
	flag, flagErr := store.runCommand(ctx, "uci", "-q", "get", "quickstart-netpolicy.main.enabled")
	var exit *exec.ExitError
	if flagErr != nil && (!errors.As(flagErr, &exit) || exit.ExitCode() != 1) {
		return errors.New("limit service configuration could not be read")
	}
	oldFlag, hadFlag := strings.TrimSpace(string(flag)), flagErr == nil
	_, enabledErr := store.runCommand(ctx, service, "enabled")
	_, runningErr := store.runCommand(ctx, service, "running")
	wasEnabled, wasRunning := enabledErr == nil, runningErr == nil
	flagChanged := false
	rollback := func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if !wasRunning {
			_, _ = store.runCommand(rollbackCtx, service, "stop")
		}
		if !wasEnabled {
			_, _ = store.runCommand(rollbackCtx, service, "disable")
		}
		if flagChanged {
			if hadFlag {
				_, _ = store.runCommand(rollbackCtx, "uci", "set", "quickstart-netpolicy.main.enabled="+oldFlag)
			} else {
				_, _ = store.runCommand(rollbackCtx, "uci", "-q", "delete", "quickstart-netpolicy.main.enabled")
			}
			_, _ = store.runCommand(rollbackCtx, "uci", "commit", "quickstart-netpolicy")
		}
	}
	if oldFlag != "1" {
		flagChanged = true
		if _, err := store.runCommand(ctx, "uci", "set", "quickstart-netpolicy.main.enabled=1"); err != nil {
			rollback()
			return errors.New("limit service could not be enabled; inspect its configuration")
		}
		if _, err := store.runCommand(ctx, "uci", "commit", "quickstart-netpolicy"); err != nil {
			rollback()
			return errors.New("limit service configuration could not be saved")
		}
	}
	if output, err := store.runCommand(ctx, service, "enable"); err != nil {
		log.Printf("capability enable failed key=%s action=enable: %s", capabilityKey, strings.TrimSpace(string(output)))
		rollback()
		return errors.New("limit service could not be enabled")
	}
	if output, err := store.runCommand(ctx, service, "start"); err != nil {
		log.Printf("capability enable failed key=%s action=start: %s", capabilityKey, strings.TrimSpace(string(output)))
		rollback()
		return errors.New("limit service could not be started; check service state before retrying")
	}
	return nil
}
