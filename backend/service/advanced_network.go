package service

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

const managementProbeTimeout = 2 * time.Second

type AdvancedNetworkModule struct {
	inventory  *DeviceInventoryModule
	groups     *DeviceGroupModule
	traffic    *TrafficInsightsModule
	audit      *NetworkAuditModule
	probeSlots chan struct{}
	now        func() time.Time
	client     *http.Client
}

func NewAdvancedNetworkModule(inventory *DeviceInventoryModule, groups *DeviceGroupModule, traffic *TrafficInsightsModule, audit *NetworkAuditModule) *AdvancedNetworkModule {
	transport := &http.Transport{
		Proxy:             nil,
		DialContext:       (&net.Dialer{Timeout: managementProbeTimeout}).DialContext,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, // Local device UIs commonly use self-signed certificates; response content is never read.
		DisableKeepAlives: true,
	}
	return &AdvancedNetworkModule{
		inventory: inventory, groups: groups, traffic: traffic, audit: audit, probeSlots: make(chan struct{}, 4), now: time.Now,
		client: &http.Client{Transport: transport, Timeout: managementProbeTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
}

func (module *AdvancedNetworkModule) Status(deviceID string) (*models.AdvancedNetworkResponse, error) {
	_, bandix := trafficInsightsCapabilities()
	insightCapability := &models.AdvancedCapability{State: "not_installed", Reason: bandix.Reason, Detail: "Optional Bandix adapter; not shown in the default device columns."}
	if bandix.State != "not_installed" {
		insightCapability = &models.AdvancedCapability{State: "unavailable", Reason: "adapter_contract_not_verified", Detail: "Bandix was detected, but this firmware has not enabled a verified connection-insights adapter."}
	}
	capabilities := map[string]*models.AdvancedCapability{
		"ipv6_reservation":       {State: "unavailable", Reason: "firmware_contract_not_verified", Detail: "IPv6 devices remain visible; no unsupported reservation control is exposed."},
		"connection_insights":    insightCapability,
		"dns_insights":           {State: "unavailable", Reason: "dns_privacy_default", Detail: "DNS names are private and are not collected or recorded by default."},
		"management_probe":       {State: "available", Detail: "LAN inventory addresses only; ports 80, 443, 8080 and 8443; four concurrent probes."},
		"segmentation":           {State: "unavailable", Reason: "segmentation_adapter_not_installed", Detail: "Guest and IoT segmentation is independent from DHCP gateway assignment."},
		"multi_wan":              {State: "unavailable", Reason: "multi_wan_adapter_not_installed", Detail: "Multi-WAN policy routing is independent from DHCP gateway assignment."},
		"identification_updates": {State: "limited", Reason: "built_in_rules_only", Detail: "Built-in source is rollback-safe; remote rule updates require a signed source contract."},
	}
	response, err := module.audit.List(deviceID, 20)
	if err != nil {
		return nil, err
	}
	response.Result.Capabilities = capabilities
	return response, nil
}

func (module *AdvancedNetworkModule) Probe(ctx context.Context, request *models.ManagementProbeRequest) (*models.ManagementProbeResponse, error) {
	started := module.now()
	address, err := module.validateProbeTarget(ctx, request)
	if err != nil {
		return managementProbeFailure("validation_failed", err.Error()), nil
	}
	select {
	case module.probeSlots <- struct{}{}:
		defer func() { <-module.probeSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	target := fmt.Sprintf("%s://%s/", request.Scheme, net.JoinHostPort(address.String(), fmt.Sprint(request.Port)))
	probeContext, cancel := context.WithTimeout(ctx, managementProbeTimeout)
	defer cancel()
	httpRequest, _ := http.NewRequestWithContext(probeContext, http.MethodHead, target, nil)
	response, requestErr := module.client.Do(httpRequest)
	result := &models.ManagementProbeResult{URL: target, DurationMS: module.now().Sub(started).Milliseconds()}
	if requestErr != nil {
		result.Error = &models.DevicePolicyError{Code: "unreachable", Message: "management interface did not respond within the safe timeout"}
		return &models.ManagementProbeResponse{Result: result}, nil
	}
	defer response.Body.Close()
	result.Reachable, result.StatusCode = true, response.StatusCode
	module.audit.Record(request.DeviceID, "management_probe", "probe", "success", "")
	return &models.ManagementProbeResponse{Result: result}, nil
}

func (module *AdvancedNetworkModule) validateProbeTarget(ctx context.Context, request *models.ManagementProbeRequest) (netip.Addr, error) {
	if request == nil || strings.TrimSpace(request.DeviceID) == "" {
		return netip.Addr{}, errors.New("deviceId is required")
	}
	if request.Scheme != "http" && request.Scheme != "https" {
		return netip.Addr{}, errors.New("scheme must be http or https")
	}
	allowedPorts := map[int]bool{80: true, 443: true, 8080: true, 8443: true}
	if !allowedPorts[request.Port] {
		return netip.Addr{}, errors.New("port is not in the management probe allowlist")
	}
	address, err := netip.ParseAddr(strings.TrimSpace(request.Address))
	if err != nil || (!address.IsPrivate() && !address.IsLinkLocalUnicast()) {
		return netip.Addr{}, errors.New("address must be a private LAN IP")
	}
	inventory, err := module.inventory.Snapshot(ctx)
	if err != nil {
		return netip.Addr{}, err
	}
	for _, device := range inventory.Result.Devices {
		if device == nil || device.DeviceID != request.DeviceID || device.Addresses == nil {
			continue
		}
		for _, candidate := range device.Addresses.Current {
			if candidate != nil && canonicalInventoryAddress(candidate.Address) == canonicalInventoryAddress(address.String()) {
				return address, nil
			}
		}
	}
	return netip.Addr{}, errors.New("address is not currently owned by this device")
}

func (module *AdvancedNetworkModule) Export(ctx context.Context) (*models.PolicyBundleResponse, error) {
	state, err := module.groups.store.Read(ctx)
	if err != nil {
		return nil, err
	}
	module.traffic.mu.Lock()
	defer module.traffic.mu.Unlock()
	if err := module.traffic.load(); err != nil {
		return nil, err
	}
	quotas := map[string]*models.TrafficQuota{}
	for id, value := range module.traffic.document.Quotas {
		copyValue := *value
		quotas[id] = &copyValue
	}
	bundle := &models.PolicyBundle{SchemaVersion: 1, Scope: "device_groups_and_quotas", ExportedAt: module.now().UTC().Format(time.RFC3339), GlobalPolicy: state.GlobalPolicy, Groups: state.Groups, DevicePolicies: state.DevicePolicies, Quotas: quotas}
	return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{Bundle: bundle}}, nil
}

func (module *AdvancedNetworkModule) ImportPlan(ctx context.Context, request *models.PolicyImportRequest) (*models.PolicyBundleResponse, error) {
	plan, policyErr := module.planImport(ctx, request)
	if policyErr != nil {
		return policyBundleFailure(policyErr.Code, policyErr.Message), nil
	}
	return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{Plan: plan}}, nil
}

func (module *AdvancedNetworkModule) ImportApply(ctx context.Context, request *models.PolicyImportRequest) (*models.PolicyBundleResponse, error) {
	plan, policyErr := module.planImport(ctx, request)
	if policyErr != nil {
		return policyBundleFailure(policyErr.Code, policyErr.Message), nil
	}
	if request == nil || !request.Confirmed {
		return policyBundleFailure("confirmation_required", "apply requires a confirmed dry-run bundle"), nil
	}
	if request.PlanChecksum == "" || request.PlanChecksum != plan.Checksum {
		return policyBundleFailure("conflict", "bundle changed after dry-run; preview it again before applying"), nil
	}
	if request.ExpectedGroupVersion == "" || request.ExpectedGroupVersion != plan.GroupVersion {
		return policyBundleFailure("conflict", "device groups changed after dry-run; preview them again before applying"), nil
	}
	groupStore, ok := module.groups.store.(*jsonDeviceGroupStore)
	if !ok {
		return policyBundleFailure("unavailable", "policy import store is unavailable"), nil
	}
	module.groups.mu.Lock()
	defer module.groups.mu.Unlock()
	module.traffic.mu.Lock()
	defer module.traffic.mu.Unlock()
	oldGroup, oldGroupRaw, err := groupStore.readDocument()
	if err != nil {
		return nil, err
	}
	if err := module.traffic.load(); err != nil {
		return nil, err
	}
	oldTraffic := cloneTrafficInsightsDocument(module.traffic.document)
	nextGroup := deviceGroupDocument{SchemaVersion: 1, GlobalPolicy: request.Bundle.GlobalPolicy, Groups: request.Bundle.Groups, DevicePolicies: request.Bundle.DevicePolicies}
	nextTraffic := cloneTrafficInsightsDocument(module.traffic.document)
	nextTraffic.Quotas = map[string]*models.TrafficQuota{}
	nextTraffic.QuotaRuntime = map[string]trafficQuotaRuntime{}
	for id, value := range request.Bundle.Quotas {
		copyValue := *value
		nextTraffic.Quotas[id] = &copyValue
	}
	if reflect.DeepEqual(oldGroup, nextGroup) && reflect.DeepEqual(oldTraffic.Quotas, nextTraffic.Quotas) {
		return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{Plan: plan, Changed: false}}, nil
	}
	updatedGroup, err := groupStore.Replace(nextGroup, request.ExpectedGroupVersion)
	if err != nil {
		return policyBundleFailure("apply_failed", err.Error()), nil
	}
	trafficSize, err := module.traffic.store.Save(&nextTraffic)
	if err != nil {
		_, rollbackErr := groupStore.Replace(oldGroup, "")
		if rollbackErr != nil {
			return policyBundleFailure("rollback_failed", err.Error()+"; rollback failed: "+rollbackErr.Error()), nil
		}
		module.traffic.document = oldTraffic
		return policyBundleFailure("apply_failed", err.Error()), nil
	}
	module.traffic.document = nextTraffic
	module.traffic.storageSize = trafficSize
	module.traffic.dirty = false
	previousGroup := documentDeviceGroupState(oldGroup, oldGroupRaw)
	module.groups.reconcileState(ctx, updatedGroup, &previousGroup, module.now())
	module.audit.Record("", "policy_bundle", "policy_changed", "success", plan.Checksum)
	return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{Plan: plan, Changed: true}}, nil
}

func (module *AdvancedNetworkModule) planImport(ctx context.Context, request *models.PolicyImportRequest) (*models.PolicyImportPlan, *models.DevicePolicyError) {
	if request == nil || request.Bundle == nil {
		return nil, &models.DevicePolicyError{Code: "validation_failed", Message: "bundle is required"}
	}
	bundle := request.Bundle
	if bundle.SchemaVersion != 1 || bundle.Scope != "device_groups_and_quotas" {
		return nil, &models.DevicePolicyError{Code: "incompatible_version", Message: "unsupported policy bundle version or scope"}
	}
	if bundle.Groups == nil || bundle.DevicePolicies == nil || bundle.Quotas == nil {
		return nil, &models.DevicePolicyError{Code: "validation_failed", Message: "bundle collections are required"}
	}
	document := deviceGroupDocument{SchemaVersion: 1, GlobalPolicy: bundle.GlobalPolicy, Groups: bundle.Groups, DevicePolicies: bundle.DevicePolicies}
	if err := validateDeviceGroupDocument(document); err != nil {
		return nil, &models.DevicePolicyError{Code: "validation_failed", Message: err.Error()}
	}
	if len(bundle.Quotas) > trafficInsightsQuotaLimit {
		return nil, &models.DevicePolicyError{Code: "capacity_exceeded", Message: "traffic quota limit reached"}
	}
	for id, quota := range bundle.Quotas {
		if quota == nil || id != quota.DeviceID {
			return nil, &models.DevicePolicyError{Code: "validation_failed", Message: "invalid quota record"}
		}
		if err := validateTrafficQuotaRequest(&models.TrafficQuotaRequest{DeviceID: quota.DeviceID, Enabled: quota.Enabled, Period: quota.Period, LimitBytes: quota.LimitBytes, Action: quota.Action}); err != nil {
			return nil, &models.DevicePolicyError{Code: "validation_failed", Message: err.Error()}
		}
	}
	current, err := module.groups.store.Read(ctx)
	if err != nil {
		return nil, &models.DevicePolicyError{Code: "read_failed", Message: err.Error()}
	}
	if request.ExpectedGroupVersion != "" && request.ExpectedGroupVersion != current.Version {
		return nil, &models.DevicePolicyError{Code: "conflict", Message: "device groups changed; export again before importing"}
	}
	raw, _ := json.Marshal(bundle)
	sum := sha256.Sum256(raw)
	warnings := []string{"Import replaces device groups and traffic quotas only; address, route, speed and firewall rules are unchanged."}
	sort.Strings(warnings)
	return &models.PolicyImportPlan{CanApply: true, GroupVersion: current.Version, GroupCount: len(bundle.Groups), DevicePolicyCount: len(bundle.DevicePolicies), QuotaCount: len(bundle.Quotas), Warnings: warnings, Checksum: hex.EncodeToString(sum[:])}, nil
}

func cloneTrafficInsightsDocument(source trafficInsightsDocument) trafficInsightsDocument {
	raw, _ := json.Marshal(source)
	var result trafficInsightsDocument
	_ = json.Unmarshal(raw, &result)
	normalizeTrafficInsightsDocument(&result)
	return result
}
func managementProbeFailure(code, message string) *models.ManagementProbeResponse {
	return &models.ManagementProbeResponse{Result: &models.ManagementProbeResult{Error: &models.DevicePolicyError{Code: code, Message: message}}}
}
func advancedNetworkFailure(code, message string) *models.AdvancedNetworkResponse {
	return &models.AdvancedNetworkResponse{Result: &models.AdvancedNetworkResult{Capabilities: map[string]*models.AdvancedCapability{}, Events: []*models.NetworkAuditEvent{}, Error: &models.DevicePolicyError{Code: code, Message: message}}}
}
func policyBundleFailure(code, message string) *models.PolicyBundleResponse {
	return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{Error: &models.DevicePolicyError{Code: code, Message: message}}}
}
