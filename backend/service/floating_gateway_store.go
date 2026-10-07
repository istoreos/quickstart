package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

const floatingGatewayConfigPath = "/etc/config/floatip"

type systemFloatingGatewayStore struct{ inventory *DeviceInventoryModule }

type floatingGatewayConfigSnapshot struct {
	Data   []byte
	Mode   os.FileMode
	Exists bool
}

var (
	floatingGatewayInstalled      = func() (bool, error) { return CheckAppIsInstalled("floatip") }
	floatingGatewayPrefixes       = readSystemLANPrefixes
	floatingGatewayRuntime        = readFloatingGatewayRuntime
	floatingGatewaySnapshotConfig = snapshotFloatingGatewayConfig
	floatingGatewayMutateConfig   = func(config *models.FloatingGatewayConfig) error {
		return mutateFloatingGatewayConfigAt(filepath.Dir(floatingGatewayConfigPath), config)
	}
	floatingGatewayReload        = reloadAndVerifyFloatingGateway
	floatingGatewayRestoreConfig = restoreFloatingGatewayConfig
)

func (store *systemFloatingGatewayStore) Read(ctx context.Context) (floatingGatewaySnapshot, error) {
	installed, err := floatingGatewayInstalled()
	if err != nil {
		return floatingGatewaySnapshot{}, err
	}
	config := &models.FloatingGatewayConfig{PeerIPs: []string{}, HealthTimeoutSec: 5}
	data, readErr := os.ReadFile(floatingGatewayConfigPath)
	version := ""
	if readErr == nil {
		sum := sha256.Sum256(data)
		version = hex.EncodeToString(sum[:])
		if loadErr := uci.LoadConfig("floatip", true); loadErr == nil {
			enabled, _ := uci.GetLast("floatip", "main", "enabled")
			role, _ := uci.GetLast("floatip", "main", "role")
			config.Enabled = enabled == "1"
			config.Role = floatingRoleToPublic(role)
			config.VirtualIP, _ = uci.GetLast("floatip", "main", "set_ip")
			config.PeerIPs, _ = uci.Get("floatip", "main", "check_ip")
			config.HealthURL, _ = uci.GetLast("floatip", "main", "check_url")
			timeout, _ := uci.GetLast("floatip", "main", "check_url_timeout")
			if parsed, parseErr := strconv.ParseInt(timeout, 10, 64); parseErr == nil && parsed > 0 {
				config.HealthTimeoutSec = parsed
			}
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return floatingGatewaySnapshot{}, readErr
	}
	config = normalizedFloatingConfig(config)
	prefixes, prefixErr := floatingGatewayPrefixes()
	if prefixErr != nil {
		prefixes = nil
	}
	running, holder := false, false
	if installed {
		running, holder = floatingGatewayRuntime(ctx, config.VirtualIP)
	}
	used := map[string]bool{}
	peerHolder := false
	if store.inventory != nil {
		if snapshot, inventoryErr := store.inventory.Snapshot(ctx); inventoryErr == nil && snapshot.Result != nil {
			peerHolder = !holder && inventoryObservesFloatingPeer(snapshot.Result.Devices, config.VirtualIP)
			for _, device := range snapshot.Result.Devices {
				if device != nil {
					for _, address := range device.Addresses.Current {
						if address != nil && address.Family == 4 {
							used[address.Address] = true
						}
					}
				}
			}
		}
	}
	return floatingGatewaySnapshot{Installed: installed, Config: config, Prefixes: prefixes, UsedIPs: used, Running: running, LocalHolder: holder, PeerHolder: peerHolder, Version: version}, nil
}

func inventoryObservesFloatingPeer(devices []*models.DeviceInventoryItem, virtualIP string) bool {
	if virtualIP == "" {
		return false
	}
	for _, device := range devices {
		if device == nil || !device.Online || device.Addresses == nil {
			continue
		}
		for _, address := range device.Addresses.Current {
			if address != nil && address.Family == 4 && address.Address == virtualIP {
				return true
			}
		}
	}
	return false
}

func (store *systemFloatingGatewayStore) Apply(ctx context.Context, config *models.FloatingGatewayConfig) error {
	snapshot, err := floatingGatewaySnapshotConfig()
	if err != nil {
		return fmt.Errorf("snapshot floating gateway configuration: %w", err)
	}
	if err := floatingGatewayMutateConfig(config); err != nil {
		return rollbackFloatingGateway(ctx, snapshot, fmt.Errorf("write floating gateway configuration: %w", err))
	}
	if err := floatingGatewayReload(ctx, config.Enabled); err != nil {
		return rollbackFloatingGateway(ctx, snapshot, fmt.Errorf("reload floating gateway: %w", err))
	}
	return nil
}

func snapshotFloatingGatewayConfig() (floatingGatewayConfigSnapshot, error) {
	info, err := os.Stat(floatingGatewayConfigPath)
	if errors.Is(err, os.ErrNotExist) {
		return floatingGatewayConfigSnapshot{}, nil
	}
	if err != nil {
		return floatingGatewayConfigSnapshot{}, err
	}
	data, err := os.ReadFile(floatingGatewayConfigPath)
	if err != nil {
		return floatingGatewayConfigSnapshot{}, err
	}
	return floatingGatewayConfigSnapshot{Data: data, Mode: info.Mode().Perm(), Exists: true}, nil
}

func mutateFloatingGatewayConfigAt(configDir string, config *models.FloatingGatewayConfig) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("floatip", true); err != nil {
		return err
	}
	sections, _ := tree.GetSections("floatip", "floatip")
	found := false
	for _, section := range sections {
		if section == "main" {
			found = true
			break
		}
	}
	if !found {
		if err := tree.AddSection("floatip", "main", "floatip"); err != nil {
			return err
		}
	}
	for _, option := range []string{"enabled", "role", "set_ip", "check_ip", "check_url", "check_url_timeout"} {
		tree.Del("floatip", "main", option)
	}
	enabled := "0"
	if config.Enabled {
		enabled = "1"
	}
	if !tree.Set("floatip", "main", "enabled", enabled) {
		return errors.New("set floating gateway enabled state")
	}
	if config.Role != "" && !tree.Set("floatip", "main", "role", floatingRoleToInternal(config.Role)) {
		return errors.New("set floating gateway responsibility")
	}
	if config.VirtualIP != "" && !tree.Set("floatip", "main", "set_ip", config.VirtualIP) {
		return errors.New("set floating gateway address")
	}
	if len(config.PeerIPs) > 0 && !tree.SetType("floatip", "main", "check_ip", uci.TypeList, config.PeerIPs...) {
		return errors.New("set peer node addresses")
	}
	if config.HealthURL != "" && !tree.Set("floatip", "main", "check_url", config.HealthURL) {
		return errors.New("set health URL")
	}
	if config.HealthTimeoutSec > 0 && !tree.Set("floatip", "main", "check_url_timeout", strconv.FormatInt(config.HealthTimeoutSec, 10)) {
		return errors.New("set health timeout")
	}
	return tree.Commit()
}

func reloadAndVerifyFloatingGateway(ctx context.Context, enabled bool) error {
	command := exec.CommandContext(ctx, "/etc/init.d/floatip", "reload")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	if !enabled {
		return nil
	}
	check := exec.CommandContext(ctx, "/etc/init.d/floatip", "running")
	if output, err := check.CombinedOutput(); err != nil {
		return fmt.Errorf("service is not running: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func restoreFloatingGatewayConfig(ctx context.Context, snapshot floatingGatewayConfigSnapshot) error {
	if snapshot.Exists {
		if err := os.WriteFile(floatingGatewayConfigPath, snapshot.Data, snapshot.Mode); err != nil {
			return err
		}
	} else if err := os.Remove(floatingGatewayConfigPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	command := exec.CommandContext(ctx, "/etc/init.d/floatip", "reload")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("restore service: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func rollbackFloatingGateway(ctx context.Context, snapshot floatingGatewayConfigSnapshot, applyErr error) error {
	if err := floatingGatewayRestoreConfig(ctx, snapshot); err != nil {
		return fmt.Errorf("%v; rollback failed: %w", applyErr, err)
	}
	return applyErr
}

func readFloatingGatewayRuntime(ctx context.Context, virtualIP string) (bool, bool) {
	running := exec.CommandContext(ctx, "/etc/init.d/floatip", "running").Run() == nil
	if virtualIP == "" {
		return running, false
	}
	output, err := exec.CommandContext(ctx, "ip", "-4", "addr", "show").Output()
	return running, err == nil && strings.Contains(string(output), " "+virtualIP+"/")
}

func floatingRoleToPublic(role string) string {
	if role == "main" {
		return "preferred"
	}
	if role == "fallback" {
		return "takeover"
	}
	return ""
}
func floatingRoleToInternal(role string) string {
	if role == "preferred" {
		return "main"
	}
	if role == "takeover" {
		return "fallback"
	}
	return ""
}
