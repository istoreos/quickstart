package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

const (
	nativePolicyProviderName = "quickstart-native"
	nativePolicyInterface    = "1"
	nativeMaximumResponse    = 1 << 20
	nativeMaximumPolicies    = 1024
)

type nativePolicyOwner struct {
	Product  string `json:"product"`
	Instance string `json:"instance"`
}

type nativeDeviceIdentity struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type nativeRateLimit struct {
	Enabled               bool  `json:"enabled"`
	UploadBitsPerSecond   int64 `json:"uploadBitsPerSecond"`
	DownloadBitsPerSecond int64 `json:"downloadBitsPerSecond"`
}

type nativePolicyInput struct {
	Device    nativeDeviceIdentity `json:"device"`
	RateLimit *nativeRateLimit     `json:"rateLimit,omitempty"`
	Quota     json.RawMessage      `json:"quota,omitempty"`
}

type nativeStoredPolicy struct {
	Revision  uint64               `json:"revision"`
	Source    string               `json:"source"`
	ID        string               `json:"id"`
	Owner     nativePolicyOwner    `json:"owner"`
	Device    nativeDeviceIdentity `json:"device"`
	RateLimit *nativeRateLimit     `json:"rateLimit,omitempty"`
	Quota     json.RawMessage      `json:"quota,omitempty"`
	State     string               `json:"state"`
}

type nativePolicySnapshot struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Revision      uint64               `json:"revision"`
	Policies      []nativeStoredPolicy `json:"policies"`
}

type nativePolicyMutation struct {
	ExpectedRevision uint64              `json:"expectedRevision"`
	IdempotencyKey   string              `json:"idempotencyKey"`
	Owner            nativePolicyOwner   `json:"owner"`
	Policies         []nativePolicyInput `json:"policies"`
}

type nativePolicyPlan struct {
	ID           string               `json:"id"`
	BaseRevision uint64               `json:"baseRevision"`
	NextRevision uint64               `json:"nextRevision"`
	Changes      []nativePolicyChange `json:"changes"`
	Warnings     []string             `json:"warnings"`
}

type nativePolicyChange struct {
	Kind   string               `json:"kind"`
	Device nativeDeviceIdentity `json:"device"`
}

type nativeEnvelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Field   string `json:"field"`
	} `json:"error"`
}

type nativeEngineHealth struct {
	State     string `json:"state"`
	Interface string `json:"interface"`
	PID       int    `json:"pid"`
	Context   *struct {
		NetworkID      string `json:"networkId"`
		InstanceID     string `json:"instanceId"`
		Epoch          string `json:"epoch"`
		InterfaceIndex int    `json:"interfaceIndex"`
	} `json:"context"`
}

type nativeRateLimitProvider struct {
	registry *nativeEngineRegistry
	binding  *nativeEngineBinding
	baseURL  string
	client   *http.Client
	owner    nativePolicyOwner
	offload  rateLimitRuntimeInspector
	now      func() time.Time
}

func NewNativeRateLimitProvider(baseURL string, client *http.Client) rateLimitProvider {
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	return &nativeRateLimitProvider{
		baseURL: strings.TrimRight(baseURL, "/"), client: client,
		owner:   nativePolicyOwner{Product: "quickstart", Instance: "local-router"},
		offload: &systemRateLimitRuntimeInspector{}, now: time.Now,
	}
}

func (*nativeRateLimitProvider) Name() string         { return nativePolicyProviderName }
func (*nativeRateLimitProvider) IdentityKind() string { return "mac" }
func (*nativeRateLimitProvider) SupportsIPv6() bool   { return true }

func (provider *nativeRateLimitProvider) Inspect(ctx context.Context, target rateLimitTarget) (rateLimitProviderObservation, error) {
	result := rateLimitProviderObservation{Offload: "unknown"}
	selected, err := provider.forContext(ctx)
	if err != nil {
		return result, err
	}
	if selected != provider {
		return selected.Inspect(ctx, target)
	}
	if err := provider.probe(ctx); err != nil {
		return result, err
	}
	snapshot, err := provider.snapshot(ctx)
	if err != nil {
		return result, err
	}
	for _, policy := range snapshot.Policies {
		if policy.Owner != provider.owner || normalizeInventoryMAC(policy.Device.Value) != normalizeInventoryMAC(target.MAC) {
			continue
		}
		result.Configured = true
		result.Loaded = policy.State == "loaded" || policy.State == "effective"
		result.Verified = policy.State == "effective"
		if policy.RateLimit != nil && policy.RateLimit.Enabled {
			result.Policy = models.DeviceSpeedPolicy{
				Enabled:       true,
				UploadSpeed:   bitsPerSecondToMbit(policy.RateLimit.UploadBitsPerSecond),
				DownloadSpeed: bitsPerSecondToMbit(policy.RateLimit.DownloadBitsPerSecond),
			}
		}
		break
	}
	if provider.offload != nil {
		if state, inspectErr := provider.offload.Offload(ctx); inspectErr == nil {
			result.Offload = state
			if state != "compatible" {
				result.Verified = false
			}
		} else {
			result.Verified = false
		}
	}
	return result, nil
}

func (provider *nativeRateLimitProvider) Apply(ctx context.Context, target rateLimitTarget, desired models.DeviceSpeedPolicy) error {
	selected, err := provider.forContext(ctx)
	if err != nil {
		return err
	}
	if selected != provider {
		return selected.Apply(ctx, target, desired)
	}
	mac := strings.ToUpper(normalizeInventoryMAC(target.MAC))
	if mac == "" {
		return errors.New("netpolicy_validation_failed: stable MAC is required")
	}
	if err := provider.probe(ctx); err != nil {
		return err
	}
	snapshot, err := provider.snapshot(ctx)
	if err != nil {
		return err
	}
	mutation := nativePolicyMutation{
		ExpectedRevision: snapshot.Revision,
		IdempotencyKey:   fmt.Sprintf("quickstart-%d-%s", provider.now().UnixNano(), strings.ReplaceAll(mac, ":", "")),
		Owner:            provider.owner,
		Policies:         make([]nativePolicyInput, 0, len(snapshot.Policies)+1),
	}
	found := false
	for _, policy := range snapshot.Policies {
		if policy.Owner != provider.owner {
			continue
		}
		input := nativePolicyInput{Device: policy.Device, RateLimit: policy.RateLimit, Quota: policy.Quota}
		if normalizeInventoryMAC(policy.Device.Value) == normalizeInventoryMAC(mac) {
			found = true
			if !desired.Enabled {
				if len(policy.Quota) > 0 {
					input.RateLimit = nil
					mutation.Policies = append(mutation.Policies, input)
				}
				continue
			}
			input.RateLimit = desiredNativeRate(desired)
		}
		mutation.Policies = append(mutation.Policies, input)
	}
	if desired.Enabled && !found {
		mutation.Policies = append(mutation.Policies, nativePolicyInput{
			Device: nativeDeviceIdentity{Kind: "mac", Value: mac}, RateLimit: desiredNativeRate(desired),
		})
	}
	if len(mutation.Policies) > nativeMaximumPolicies {
		return errors.New("netpolicy_capacity_exceeded")
	}

	var plan nativePolicyPlan
	if err := provider.call(ctx, http.MethodPost, "/api/v1/policies/plan", mutation, &plan); err != nil {
		return err
	}
	if plan.Changes == nil || plan.Warnings == nil {
		return errors.New("netpolicy_incompatible_plan")
	}
	var committed nativePolicySnapshot
	if err := provider.call(ctx, http.MethodPost, "/api/v1/policies/apply", map[string]any{"plan": plan, "request": mutation}, &committed); err != nil {
		return err
	}
	var verified nativePolicySnapshot
	if err := provider.call(ctx, http.MethodPost, "/api/v1/policies/verify", committed, &verified); err != nil {
		return fmt.Errorf("netpolicy_verification_failed: %w", err)
	}
	return nil
}

func desiredNativeRate(desired models.DeviceSpeedPolicy) *nativeRateLimit {
	return &nativeRateLimit{
		Enabled: true, UploadBitsPerSecond: desired.UploadSpeed * 1_000_000,
		DownloadBitsPerSecond: desired.DownloadSpeed * 1_000_000,
	}
}

func bitsPerSecondToMbit(value int64) int64 {
	if value <= 0 {
		return 0
	}
	return (value + 999_999) / 1_000_000
}

func (provider *nativeRateLimitProvider) probe(ctx context.Context) error {
	selected, err := provider.forContext(ctx)
	if err != nil {
		return err
	}
	if selected != provider {
		return selected.probe(ctx)
	}
	var capabilities struct {
		Product             string `json:"product"`
		InterfaceVersion    string `json:"interfaceVersion"`
		Engine              string `json:"engine"`
		ScopeContextVersion int    `json:"scopeContextVersion"`
	}
	if err := provider.call(ctx, http.MethodGet, "/api/v1/system/capabilities", nil, &capabilities); err != nil {
		return err
	}
	if capabilities.Product != "quickstart-netpolicy" || capabilities.InterfaceVersion != nativePolicyInterface || capabilities.Engine != "quickstart-native" {
		return errors.New("netpolicy_incompatible_interface")
	}
	var health nativeEngineHealth
	if err := provider.call(ctx, http.MethodGet, "/api/v1/system/health", nil, &health); err != nil {
		return err
	}
	if provider.binding != nil {
		binding := provider.binding
		if capabilities.ScopeContextVersion >= 1 {
			if health.Context == nil || health.Interface != binding.Interface || health.Context.NetworkID != binding.NetworkID || health.Context.InstanceID != binding.InstanceID || health.Context.Epoch == "" || health.Context.InterfaceIndex != binding.InterfaceIndex {
				return errors.New("netpolicy_runtime_binding_mismatch")
			}
		} else {
			// Old v1 engines expose no context. Permit only the original main LAN and
			// verify the local process's interface before retaining legacy behavior.
			if binding.NetworkID != defaultLAN || binding.InstanceID != "main" || !legacyNativeProcessMatches(health.PID, binding.Interface, binding.Port) {
				return errors.New("netpolicy_scope_interface_unsupported")
			}
		}
	}
	if health.State != "healthy" {
		return fmt.Errorf("netpolicy_unhealthy: %s", health.State)
	}
	return nil
}

func (provider *nativeRateLimitProvider) snapshot(ctx context.Context) (nativePolicySnapshot, error) {
	selected, selectErr := provider.forContext(ctx)
	if selectErr != nil {
		return nativePolicySnapshot{}, selectErr
	}
	if selected != provider {
		return selected.snapshot(ctx)
	}
	var snapshot nativePolicySnapshot
	err := provider.call(ctx, http.MethodGet, "/api/v1/policies", nil, &snapshot)
	if len(snapshot.Policies) > nativeMaximumPolicies {
		return nativePolicySnapshot{}, errors.New("netpolicy_response_capacity_exceeded")
	}
	return snapshot, err
}

func (provider *nativeRateLimitProvider) replaceOwned(
	ctx context.Context,
	before nativePolicySnapshot,
	policies []nativePolicyInput,
	idempotencyKey string,
) (nativePolicySnapshot, error) {
	selected, selectErr := provider.forContext(ctx)
	if selectErr != nil {
		return nativePolicySnapshot{}, selectErr
	}
	if selected != provider {
		return selected.replaceOwned(ctx, before, policies, idempotencyKey)
	}
	if err := provider.probe(ctx); err != nil {
		return nativePolicySnapshot{}, err
	}
	mutation := nativePolicyMutation{
		ExpectedRevision: before.Revision,
		IdempotencyKey:   idempotencyKey,
		Owner:            provider.owner,
		Policies:         policies,
	}
	var plan nativePolicyPlan
	if err := provider.call(ctx, http.MethodPost, "/api/v1/policies/plan", mutation, &plan); err != nil {
		return nativePolicySnapshot{}, err
	}
	if plan.Changes == nil || plan.Warnings == nil {
		return nativePolicySnapshot{}, errors.New("netpolicy_incompatible_plan")
	}
	var committed nativePolicySnapshot
	if err := provider.call(ctx, http.MethodPost, "/api/v1/policies/apply", map[string]any{"plan": plan, "request": mutation}, &committed); err != nil {
		return nativePolicySnapshot{}, err
	}
	var verified nativePolicySnapshot
	if err := provider.call(ctx, http.MethodPost, "/api/v1/policies/verify", committed, &verified); err != nil {
		return nativePolicySnapshot{}, fmt.Errorf("netpolicy_verification_failed: %w", err)
	}
	return committed, nil
}

func (provider *nativeRateLimitProvider) ownedInputs(snapshot nativePolicySnapshot) []nativePolicyInput {
	inputs := make([]nativePolicyInput, 0, len(snapshot.Policies))
	for _, policy := range snapshot.Policies {
		if policy.Owner == provider.owner {
			inputs = append(inputs, nativePolicyInput{Device: policy.Device, RateLimit: policy.RateLimit, Quota: policy.Quota})
		}
	}
	return inputs
}

func (provider *nativeRateLimitProvider) call(ctx context.Context, method, path string, payload any, result any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, provider.baseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	// Original v1 engines serve one request per TCP connection without an
	// explicit close response header. Never reuse that half-closed socket.
	request.Close = true
	response, err := provider.client.Do(request)
	if err != nil {
		return fmt.Errorf("netpolicy_unavailable: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, nativeMaximumResponse+1))
	if err != nil {
		return err
	}
	if len(data) > nativeMaximumResponse {
		return errors.New("netpolicy_response_too_large")
	}
	var envelope nativeEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return errors.New("netpolicy_invalid_response")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || envelope.Status != "success" {
		if envelope.Error != nil && envelope.Error.Code != "" {
			return fmt.Errorf("netpolicy_%s: %s", envelope.Error.Code, envelope.Error.Message)
		}
		return fmt.Errorf("netpolicy_status_%d", response.StatusCode)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, result); err != nil {
		return errors.New("netpolicy_invalid_response")
	}
	return nil
}

func defaultNativePolicyBaseURL() string { return "http://127.0.0.1:8765" }

func nativeProviderCapability(ctx context.Context) (installed bool, available bool, reason string) {
	for _, path := range []string{"/opt/quickstart-netpolicy/current/quickstart-netpolicy", "/usr/bin/quickstart-netpolicy"} {
		if _, err := os.Stat(path); err == nil {
			installed = true
			break
		}
	}
	provider := newScopedNativeRateLimitProvider()
	provider.offload = nil
	if err := provider.probe(ctx); err != nil {
		if strings.Contains(err.Error(), "incompatible") {
			return installed, false, "interface_not_supported"
		}
		if installed {
			return true, false, "engine_unavailable"
		}
		return false, false, "dependency_not_installed"
	}
	return true, true, ""
}

func legacyNativeProcessMatches(pid int, iface string, port int) bool {
	if pid <= 0 {
		return false
	}
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return false
	}
	args := strings.Split(strings.TrimRight(string(raw), "\x00"), "\x00")
	if len(args) == 0 || filepath.Base(args[0]) != "quickstart-netpolicy" {
		return false
	}
	foundInterface, foundPort := "", "8765"
	trafficEnabled := false
	for i, arg := range args {
		if arg == "--enable-traffic" || arg == "--enable-traffic=true" {
			trafficEnabled = true
		}
		if arg == "--iface" && i+1 < len(args) {
			foundInterface = args[i+1]
		}
		if strings.HasPrefix(arg, "--iface=") {
			foundInterface = strings.TrimPrefix(arg, "--iface=")
		}
		if arg == "--port" && i+1 < len(args) {
			foundPort = args[i+1]
		}
		if strings.HasPrefix(arg, "--port=") {
			foundPort = strings.TrimPrefix(arg, "--port=")
		}
	}
	parsedPort, _ := strconv.Atoi(foundPort)
	return foundInterface == iface && parsedPort == port && trafficEnabled
}
