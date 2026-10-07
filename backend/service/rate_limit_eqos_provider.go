package service

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

// eqosRateLimitProvider adapts the legacy IP-based eqos service to the product
// rate-limit domain. No caller outside this adapter needs to know UCI sections,
// init scripts, IFB devices or tc output.
type eqosRateLimitProvider struct {
	configDir string
	runtime   rateLimitRuntimeInspector
}

func newEqosRateLimitProvider(configDir string, runtime rateLimitRuntimeInspector) rateLimitProvider {
	return &eqosRateLimitProvider{configDir: configDir, runtime: runtime}
}

func (provider *eqosRateLimitProvider) Name() string         { return "eqos" }
func (provider *eqosRateLimitProvider) IdentityKind() string { return "ipv4" }
func (provider *eqosRateLimitProvider) SupportsIPv6() bool   { return false }

func (provider *eqosRateLimitProvider) Inspect(ctx context.Context, target rateLimitTarget) (rateLimitProviderObservation, error) {
	result := rateLimitProviderObservation{Offload: "unknown"}
	addresses := []string{target.CurrentIPv4}
	if target.ReservedIP != "" && target.ReservedIP != target.CurrentIPv4 {
		addresses = append(addresses, target.ReservedIP)
	}
	for _, address := range addresses {
		if strings.TrimSpace(address) == "" {
			continue
		}
		snapshot, err := readDeviceSpeedPolicyAt(provider.configDir, address)
		if err != nil {
			return result, err
		}
		if snapshot != nil && snapshot.Section != "" {
			result.Configured = true
			result.Policy = models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: snapshot.Upload, DownloadSpeed: snapshot.Download}
			break
		}
	}
	if provider.runtime == nil {
		return result, nil
	}
	loaded, err := provider.runtime.Loaded(ctx, target.CurrentIPv4)
	if err == nil {
		result.Loaded = result.Configured && loaded
		result.Verified = result.Loaded
	}
	if offload, offloadErr := provider.runtime.Offload(ctx); offloadErr == nil {
		result.Offload = offload
	}
	return result, nil
}

func (provider *eqosRateLimitProvider) Apply(ctx context.Context, target rateLimitTarget, desired models.DeviceSpeedPolicy) error {
	address := stableRateLimitIPv4(target)
	if !desired.Enabled {
		address = strings.TrimSpace(target.CurrentIPv4)
		if address == "" {
			address = strings.TrimSpace(target.ReservedIP)
		}
	}
	if address == "" {
		return errors.New("rate limit IPv4 address is unavailable")
	}
	if err := writeDeviceSpeedPolicyAt(provider.configDir, address, target.MAC, target.DisplayName, desired.Enabled, desired.UploadSpeed, desired.DownloadSpeed); err != nil {
		return err
	}
	return deviceRestrictionApply(ctx, []string{"eqos"})
}

type systemRateLimitRuntimeInspector struct{}

var rateLimitCommandOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func (*systemRateLimitRuntimeInspector) Loaded(ctx context.Context, _ string) (bool, error) {
	if _, err := rateLimitCommandOutput(ctx, "/etc/init.d/eqos", "running"); err != nil {
		return false, nil
	}
	output, err := rateLimitCommandOutput(ctx, "tc", "qdisc", "show", "dev", "br-lan")
	if err != nil {
		return false, err
	}
	text := strings.ToLower(string(output))
	return strings.Contains(text, "htb") || strings.Contains(text, "ingress"), nil
}

func (*systemRateLimitRuntimeInspector) Offload(_ context.Context) (string, error) {
	tree := uci.NewTree(filepath.Dir(dhcpConfigPath))
	if err := tree.LoadConfig("firewall", true); err != nil {
		return "unknown", err
	}
	software, _ := tree.GetLast("firewall", "@defaults[0]", "flow_offloading")
	hardware, _ := tree.GetLast("firewall", "@defaults[0]", "flow_offloading_hw")
	if software == "1" || hardware == "1" {
		return "risk", nil
	}
	return "compatible", nil
}
