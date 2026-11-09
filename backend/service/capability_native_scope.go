package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const nativeScopeContract = "quickstart-scope-context: 1"

func (store *systemCapabilityActionStore) nativeConfigDir() string {
	if store.configDir != "" {
		return store.configDir
	}
	return "/etc/config"
}
func (store *systemCapabilityActionStore) nativeService() string {
	if store.servicePath != "" {
		return store.servicePath
	}
	return "/etc/init.d/quickstart-netpolicy"
}
func (store *systemCapabilityActionStore) fileInfo(path string) (os.FileInfo, error) {
	if store.stat != nil {
		return store.stat(path)
	}
	return os.Stat(path)
}
func (store *systemCapabilityActionStore) componentInstalled(key string) (bool, error) {
	if key == "device_speed_limit" {
		for _, path := range []string{"/opt/quickstart-netpolicy/current/quickstart-netpolicy", "/usr/bin/quickstart-netpolicy"} {
			if _, err := store.fileInfo(path); err == nil {
				return true, nil
			}
		}
	}
	pkg := capabilityInstallTarget(key)
	if key == "floating_gateway" {
		pkg = "floatip"
	}
	return componentPackageInstalled(pkg, store.fileInfo)
}
func (store *systemCapabilityActionStore) hasScopeContract() bool {
	raw, err := os.ReadFile(store.nativeService())
	return err == nil && strings.Contains(string(raw), nativeScopeContract)
}
func nativeSectionNetwork(tree uci.Tree, section string) string {
	network, _ := tree.GetLast("quickstart-netpolicy", section, "network")
	if network == "" && section == "main" {
		network = defaultLAN
	}
	return network
}

func (store *systemCapabilityActionStore) scopedNativeCapability(ctx context.Context) (*models.Capability, error) {
	installed, err := store.componentInstalled("device_speed_limit")
	if err != nil {
		return &models.Capability{State: "error", Reason: "installation_state_unavailable"}, nil
	}
	if !installed {
		return &models.Capability{State: "not_installed", Reason: "dependency_not_installed", Actions: []*models.CapabilityAction{{Kind: "install", Target: "quickstart-netpolicy", RequiresConfirmation: true}}}, nil
	}
	scope := selectedLANScope(ctx)
	if scope.ReadOnlyReason != "" {
		return &models.Capability{State: "unsupported", Reason: scope.ReadOnlyReason}, nil
	}
	if !store.hasScopeContract() {
		if scope.NetworkID == defaultLAN {
			return nativeComponentCapability(ctx, nativeProviderCapability, store.runCommand)
		}
		return &models.Capability{State: "unsupported", Reason: "multi_lan_service_upgrade_required"}, nil
	}
	disabled := func(reason string) *models.Capability {
		return &models.Capability{State: "disabled", Reason: reason, Actions: []*models.CapabilityAction{{Kind: "enable", RequiresConfirmation: true}}}
	}
	tree := uci.NewTree(store.nativeConfigDir())
	if err := tree.LoadConfig("quickstart-netpolicy", true); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return disabled("network_not_connected"), nil
		}
		return nil, err
	}
	sections, _ := tree.GetSections("quickstart-netpolicy", "quickstart-netpolicy")
	enabled := 0
	for _, section := range sections {
		if nativeSectionNetwork(tree, section) == scope.NetworkID {
			flag, _ := tree.GetLast("quickstart-netpolicy", section, "enabled")
			if flag == "1" || flag == "true" || flag == "yes" || flag == "on" {
				enabled++
			}
		}
	}
	if enabled == 0 {
		return disabled("network_not_connected"), nil
	}
	if enabled > 1 {
		return &models.Capability{State: "error", Reason: "netpolicy_duplicate_network_binding"}, nil
	}
	provider := newScopedNativeRateLimitProvider()
	provider.registry = &nativeEngineRegistry{configDir: store.nativeConfigDir()}
	if err := provider.probe(ctx); err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return disabled("network_service_stopped"), nil
		}
		return &models.Capability{State: "error", Reason: "network_service_unavailable", Actions: []*models.CapabilityAction{{Kind: "retry"}}}, nil
	}
	return &models.Capability{State: "available"}, nil
}

func (store *systemCapabilityActionStore) enableNativeScope(ctx context.Context, key string) error {
	if key != "device_speed_limit" {
		return errors.New("this component cannot be connected to this network")
	}
	scope := selectedLANScope(ctx)
	if scope == nil {
		return errors.New("network is required")
	}
	if !store.hasScopeContract() {
		if scope.NetworkID == defaultLAN {
			return store.enableLegacyNative(ctx, key)
		}
		return errors.New("请在软件中心升级限速组件后重试")
	}
	locked, release, err := lockLANWrite(ctx)
	if err != nil {
		return err
	}
	defer release()
	ctx = locked
	dir := store.nativeConfigDir()
	path := filepath.Join(dir, "quickstart-netpolicy")
	raw, readErr := os.ReadFile(path)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	existed := readErr == nil
	_, bootErr := store.runCommand(ctx, store.nativeService(), "enabled")
	if bootErr != nil {
		var exit *exec.ExitError
		if !errors.As(bootErr, &exit) || exit.ExitCode() != 1 {
			return errors.New("无法确认限速服务的启动状态")
		}
	}
	restore := func(cause error) error {
		recovery, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		var restoreErr error
		if existed {
			restoreErr = persistClassificationOverrides(path, raw)
		} else {
			restoreErr = os.Remove(path)
			if errors.Is(restoreErr, os.ErrNotExist) {
				restoreErr = nil
			}
		}
		_, reloadErr := store.runCommand(recovery, store.nativeService(), "reload")
		if bootErr != nil {
			_, _ = store.runCommand(recovery, store.nativeService(), "disable")
		}
		if restoreErr != nil || reloadErr != nil {
			return &networkRecoveryError{fmt.Errorf("component recovery_required: %v; %v; %v", cause, restoreErr, reloadErr)}
		}
		return cause
	}
	if !existed {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, nil, 0600); err != nil {
			return err
		}
	}
	tree := uci.NewTree(dir)
	if err := tree.LoadConfig("quickstart-netpolicy", true); err != nil {
		return restore(err)
	}
	section := ""
	sections, _ := tree.GetSections("quickstart-netpolicy", "quickstart-netpolicy")
	for _, candidate := range sections {
		if nativeSectionNetwork(tree, candidate) == scope.NetworkID {
			if section != "" {
				return restore(errors.New("duplicate network binding"))
			}
			section = candidate
		}
	}
	if section == "" {
		section = "main"
		if scope.NetworkID != defaultLAN {
			sum := sha256.Sum256([]byte(scope.NetworkID))
			section = "qs_" + hex.EncodeToString(sum[:6])
		}
		if err := tree.AddSection("quickstart-netpolicy", section, "quickstart-netpolicy"); err != nil {
			return restore(err)
		}
	}
	port, _ := tree.GetLast("quickstart-netpolicy", section, "port")
	dataDir, _ := tree.GetLast("quickstart-netpolicy", section, "data_dir")
	if section == "main" {
		if port == "" {
			port = "8765"
		}
		if dataDir == "" {
			dataDir = "/usr/share/quickstart-netpolicy"
		}
	}
	if port == "" {
		used := map[int]bool{8765: true}
		for _, other := range sections {
			value, _ := tree.GetLast("quickstart-netpolicy", other, "port")
			number, _ := strconv.Atoi(value)
			used[number] = true
		}
		for candidate := 8766; candidate < 8866; candidate++ {
			if used[candidate] {
				continue
			}
			listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", candidate))
			if err == nil {
				listener.Close()
				port = strconv.Itoa(candidate)
				break
			}
		}
		if port == "" {
			return restore(errors.New("当前没有可用的限速服务入口"))
		}
	}
	if dataDir == "" {
		dataDir = "/usr/share/quickstart-netpolicy-" + scope.NetworkID
	}
	for option, value := range map[string]string{"enabled": "1", "network": scope.NetworkID, "interface": scope.Device, "port": port, "data_dir": dataDir} {
		if !tree.Set("quickstart-netpolicy", section, option, value) {
			return restore(errors.New("save network connection"))
		}
	}
	if err := tree.Commit(); err != nil {
		return restore(err)
	}
	provider := newScopedNativeRateLimitProvider()
	provider.registry = &nativeEngineRegistry{configDir: dir}
	if _, err := provider.forContext(ctx); err != nil {
		return restore(err)
	}
	if _, err := store.runCommand(ctx, store.nativeService(), "enable"); err != nil {
		return restore(err)
	}
	if _, err := store.runCommand(ctx, store.nativeService(), "reload"); err != nil {
		return restore(err)
	}
	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := provider.probe(ctx); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return restore(ctx.Err())
		case <-deadline.C:
			return restore(errors.New("当前网络的限速服务未就绪，已恢复原配置"))
		case <-tick.C:
		}
	}
}

func (store *systemCapabilityActionStore) enabledNetworks(selected string) []string {
	networks := map[string]bool{selected: true}
	tree := uci.NewTree(store.nativeConfigDir())
	if tree.LoadConfig("quickstart-netpolicy", true) == nil {
		sections, _ := tree.GetSections("quickstart-netpolicy", "quickstart-netpolicy")
		for _, section := range sections {
			flag, _ := tree.GetLast("quickstart-netpolicy", section, "enabled")
			if flag == "1" || flag == "true" || flag == "yes" || flag == "on" {
				if network := nativeSectionNetwork(tree, section); network != "" {
					networks[network] = true
				}
			}
		}
	}
	result := make([]string, 0, len(networks))
	for network := range networks {
		result = append(result, network)
	}
	sort.Strings(result)
	return result
}
