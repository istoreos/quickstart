package service

import "github.com/istoreos/quickstart/backend/models"

// AdvancedNetworkModule is a read-oriented facade. Active probing and policy
// bundle mutation live in dedicated modules so their lifecycles stay isolated.
type AdvancedNetworkModule struct {
	audit      *NetworkAuditModule
	management *ManagementProbeModule
	bundles    *PolicyBundleModule
}

func NewAdvancedNetworkModule(inventory *DeviceInventoryModule, groups *DeviceGroupModule, traffic *TrafficInsightsModule, audit *NetworkAuditModule) *AdvancedNetworkModule {
	return &AdvancedNetworkModule{
		audit:      audit,
		management: NewManagementProbeModule(inventory, audit),
		bundles:    NewPolicyBundleModule(groups, traffic, audit),
	}
}

func (module *AdvancedNetworkModule) Status(deviceID string) (*models.AdvancedNetworkResponse, error) {
	_, bandix := trafficInsightsCapabilities()
	insightCapability := &models.AdvancedCapability{State: models.CapabilityNotInstalled, Reason: bandix.Reason, Detail: "Optional Bandix adapter; not shown in the default device columns."}
	if bandix.State != models.CapabilityNotInstalled {
		insightCapability = &models.AdvancedCapability{State: models.CapabilityUnsupported, Reason: "adapter_contract_not_verified", Detail: "Bandix was detected, but this firmware has not enabled a verified connection-insights adapter."}
	}
	capabilities := map[string]*models.AdvancedCapability{
		"ipv6_reservation":       {State: models.CapabilityUnsupported, Reason: "firmware_contract_not_verified", Detail: "IPv6 devices remain visible; no unsupported reservation control is exposed."},
		"connection_insights":    insightCapability,
		"dns_insights":           {State: models.CapabilityUnsupported, Reason: "dns_privacy_default", Detail: "DNS names are private and are not collected or recorded by default."},
		"management_probe":       {State: models.CapabilityAvailable, Detail: "LAN inventory addresses only; ports 80, 443, 8080 and 8443; four concurrent probes."},
		"segmentation":           {State: models.CapabilityUnsupported, Reason: "segmentation_adapter_not_installed", Detail: "Guest and IoT segmentation is independent from DHCP gateway assignment."},
		"multi_wan":              {State: models.CapabilityUnsupported, Reason: "multi_wan_adapter_not_installed", Detail: "Multi-WAN policy routing is independent from DHCP gateway assignment."},
		"identification_updates": {State: models.CapabilityAvailable, Reason: "built_in_rules_only", Detail: "Built-in source is rollback-safe; remote rule updates require a signed source contract."},
	}
	response, err := module.audit.List(deviceID, 20)
	if err != nil {
		return nil, err
	}
	response.Result.Capabilities = capabilities
	return response, nil
}

func advancedNetworkFailure(code, message string) *models.AdvancedNetworkResponse {
	return &models.AdvancedNetworkResponse{Result: &models.AdvancedNetworkResult{Capabilities: map[string]*models.AdvancedCapability{}, Events: []*models.NetworkAuditEvent{}, Error: &models.DevicePolicyError{Code: code, Message: message}}}
}
