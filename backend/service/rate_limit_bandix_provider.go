package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

const (
	bandixSchedulePath           = "/api/traffic/limits/schedule"
	bandixMaximumResponseBytes   = 1 << 20
	bandixMaximumScheduleRules   = 512
	defaultRateLimitProviderPath = "/etc/quickstart/rate-limit-provider"
)

type bandixScheduleRule struct {
	ID       any    `json:"id,omitempty"`
	MAC      string `json:"mac"`
	TimeSlot struct {
		Start string `json:"start"`
		End   string `json:"end"`
		Days  []int  `json:"days"`
	} `json:"time_slot"`
	UploadBytes   int64    `json:"wan_tx_rate_limit"`
	DownloadBytes int64    `json:"wan_rx_rate_limit"`
	UnknownFields []string `json:"-"`
}

func (rule *bandixScheduleRule) UnmarshalJSON(data []byte) error {
	type alias bandixScheduleRule
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, known := range []string{"id", "mac", "time_slot", "wan_tx_rate_limit", "wan_rx_rate_limit"} {
		delete(fields, known)
	}
	decoded.UnknownFields = make([]string, 0, len(fields))
	for field := range fields {
		decoded.UnknownFields = append(decoded.UnknownFields, field)
	}
	sort.Strings(decoded.UnknownFields)
	*rule = bandixScheduleRule(decoded)
	return nil
}

type bandixRateLimitProvider struct {
	endpoint string
	client   *http.Client
	runtime  rateLimitRuntimeInspector
}

func NewBandixRateLimitProvider(baseURL string, client *http.Client) rateLimitProvider {
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	return &bandixRateLimitProvider{endpoint: strings.TrimRight(baseURL, "/") + bandixSchedulePath, client: client, runtime: &systemRateLimitRuntimeInspector{}}
}

func (*bandixRateLimitProvider) Name() string         { return "bandix" }
func (*bandixRateLimitProvider) IdentityKind() string { return "mac" }
func (*bandixRateLimitProvider) SupportsIPv6() bool   { return true }

func (provider *bandixRateLimitProvider) Inspect(ctx context.Context, target rateLimitTarget) (rateLimitProviderObservation, error) {
	result := rateLimitProviderObservation{Offload: "unknown"}
	rules, err := provider.rules(ctx)
	if err != nil {
		return result, err
	}
	for _, rule := range rules {
		if normalizeInventoryMAC(rule.MAC) != normalizeInventoryMAC(target.MAC) {
			continue
		}
		result.Configured, result.Loaded, result.Verified = true, true, true
		result.Policy = models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: bytesPerSecondToMbit(rule.UploadBytes), DownloadSpeed: bytesPerSecondToMbit(rule.DownloadBytes)}
		break
	}
	if provider.runtime != nil {
		if offload, offloadErr := provider.runtime.Offload(ctx); offloadErr == nil {
			result.Offload = offload
		}
	}
	return result, nil
}

func (provider *bandixRateLimitProvider) Apply(ctx context.Context, target rateLimitTarget, desired models.DeviceSpeedPolicy) error {
	if strings.TrimSpace(target.MAC) == "" {
		return errors.New("bandix requires a stable MAC address")
	}
	rules, err := provider.rules(ctx)
	if err != nil {
		return err
	}
	var existing *bandixScheduleRule
	for index := range rules {
		if normalizeInventoryMAC(rules[index].MAC) == normalizeInventoryMAC(target.MAC) {
			existing = &rules[index]
			break
		}
	}
	if !desired.Enabled {
		if existing == nil {
			return nil
		}
		return provider.request(ctx, http.MethodDelete, map[string]any{"id": existing.ID})
	}
	payload := map[string]any{
		"mac":               strings.ToUpper(normalizeInventoryMAC(target.MAC)),
		"time_slot":         map[string]any{"start": "00:00", "end": "23:59", "days": []int{1, 2, 3, 4, 5, 6, 7}},
		"wan_tx_rate_limit": mbitToBytesPerSecond(desired.UploadSpeed),
		"wan_rx_rate_limit": mbitToBytesPerSecond(desired.DownloadSpeed),
	}
	method := http.MethodPost
	if existing != nil {
		method, payload["id"] = http.MethodPut, existing.ID
	}
	return provider.request(ctx, method, payload)
}

func (provider *bandixRateLimitProvider) rules(ctx context.Context) ([]bandixScheduleRule, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.endpoint, nil)
	if err != nil {
		return nil, err
	}
	response, err := provider.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("bandix_unavailable: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("bandix_status_%d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, bandixMaximumResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > bandixMaximumResponseBytes {
		return nil, errors.New("bandix_response_too_large")
	}
	var rules []bandixScheduleRule
	if err := json.Unmarshal(data, &rules); err != nil {
		var envelope struct {
			Status  string               `json:"status"`
			Data    json.RawMessage      `json:"data"`
			Rules   []bandixScheduleRule `json:"rules"`
			Message string               `json:"message"`
		}
		if envelopeErr := json.Unmarshal(data, &envelope); envelopeErr != nil {
			return nil, errors.New("bandix_invalid_response")
		}
		if envelope.Status != "" && envelope.Status != "success" {
			if envelope.Message == "" {
				envelope.Message = "provider rejected request"
			}
			return nil, fmt.Errorf("bandix_rejected: %s", envelope.Message)
		}
		rules = envelope.Rules
		if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
			if dataErr := json.Unmarshal(envelope.Data, &rules); dataErr != nil {
				var payload struct {
					Limits []bandixScheduleRule `json:"limits"`
				}
				if payloadErr := json.Unmarshal(envelope.Data, &payload); payloadErr != nil || payload.Limits == nil {
					return nil, errors.New("bandix_invalid_response")
				}
				rules = payload.Limits
			}
		}
	}
	if len(rules) > bandixMaximumScheduleRules {
		return nil, errors.New("bandix_rule_limit_exceeded")
	}
	return rules, nil
}

func (provider *bandixRateLimitProvider) request(ctx context.Context, method string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, method, provider.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := provider.client.Do(request)
	if err != nil {
		return fmt.Errorf("bandix_unavailable: %w", err)
	}
	defer response.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if readErr != nil {
		return readErr
	}
	if len(data) > 64<<10 {
		return errors.New("bandix_response_too_large")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("bandix_status_%d", response.StatusCode)
	}
	if len(bytes.TrimSpace(data)) > 0 {
		var envelope struct {
			Status  string `json:"status"`
			Success *bool  `json:"success"`
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &envelope) == nil {
			if (envelope.Status != "" && envelope.Status != "success") || (envelope.Success != nil && !*envelope.Success) {
				if envelope.Error == "" {
					envelope.Error = envelope.Message
				}
				if envelope.Error == "" {
					envelope.Error = "provider rejected request"
				}
				return fmt.Errorf("bandix_rejected: %s", envelope.Error)
			}
		}
	}
	return nil
}

func mbitToBytesPerSecond(value int64) int64 { return value * 125000 }
func bytesPerSecondToMbit(value int64) int64 {
	if value <= 0 {
		return 0
	}
	return (value + 124999) / 125000
}

func defaultBandixBaseURL() string {
	port := "8686"
	if data, err := os.ReadFile("/etc/config/bandix"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[0] == "option" && fields[1] == "port" {
				candidate := strings.Trim(fields[2], "'\"")
				if parsed, parseErr := strconv.Atoi(candidate); parseErr == nil && parsed > 0 && parsed <= 65535 {
					port = candidate
				}
			}
		}
	}
	return (&url.URL{Scheme: "http", Host: "127.0.0.1:" + port}).String()
}

func readRateLimitProviderPreference(path string) string {
	data, err := os.ReadFile(path)
	if err == nil {
		value := strings.TrimSpace(string(data))
		if value == "bandix" || value == nativePolicyProviderName {
			return value
		}
	}
	return "eqos"
}

func effectiveRateLimitProvider(path string) string {
	preferred := readRateLimitProviderPreference(path)
	if preferred == nativePolicyProviderName {
		// Never silently fall back: doing so would leave native rules configured
		// while writing a second provider's state.
		return nativePolicyProviderName
	}
	if preferred != "bandix" {
		return "eqos"
	}
	installed, supported, _ := bandixProviderCapability()
	if !installed || !supported {
		return "eqos"
	}
	return "bandix"
}

func writeRateLimitProviderPreference(path, provider string) error {
	if provider != "eqos" && provider != "bandix" && provider != nativePolicyProviderName {
		return errors.New("unsupported rate limit provider")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, []byte(provider+"\n"), 0600); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

func bandixProviderCapability() (installed bool, supported bool, reason string) {
	installed, _ = CheckAppIsInstalled("bandix")
	if !installed {
		installed, _ = CheckAppIsInstalled("luci-app-bandix")
	}
	supported = bandixKernelSupported("/proc/sys/kernel/osrelease")
	if !installed {
		return false, supported, "dependency_not_installed"
	}
	if !supported {
		return true, false, "kernel_not_supported"
	}
	return true, true, ""
}

func bandixKernelSupported(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	majorText := strings.SplitN(strings.TrimSpace(string(data)), ".", 2)[0]
	major, err := strconv.Atoi(majorText)
	return err == nil && major >= 6
}
