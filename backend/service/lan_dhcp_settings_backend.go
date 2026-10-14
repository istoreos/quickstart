package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/istoreos/quickstart/backend/models"
)

func (backend *ServiceBackend) GetLanDHCPSettingsV2(ctx context.Context) (*models.LanDHCPSettingsResponse, error) {
	return backend.lanDHCPSettingsModule().Get(ctx)
}

func (backend *ServiceBackend) PostLanDHCPSettingsPlanV2(ctx context.Context, request *http.Request) (*models.LanDHCPSettingsResponse, error) {
	var input models.LanDHCPSettingsApplyRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		return invalidLanDHCPSettingsResponse(), nil
	}
	return backend.lanDHCPSettingsModule().Plan(ctx, &input)
}

func (backend *ServiceBackend) PostLanDHCPSettingsApplyV2(ctx context.Context, request *http.Request) (*models.LanDHCPSettingsResponse, error) {
	var input models.LanDHCPSettingsApplyRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		return invalidLanDHCPSettingsResponse(), nil
	}
	response, err := backend.lanDHCPSettingsModule().Apply(ctx, &input)
	if err == nil && response != nil && response.Result != nil && response.Result.Error == nil && response.Result.Changed {
		backend.auditModule().Record("", "lan_dhcp", "policy_changed", "success", "settings")
	}
	return response, err
}

func invalidLanDHCPSettingsResponse() *models.LanDHCPSettingsResponse {
	return &models.LanDHCPSettingsResponse{Result: &models.LanDHCPSettingsResult{
		Conflicts: []*models.LanDHCPConflict{}, ReloadServices: []string{},
		Error: &models.DevicePolicyError{Code: "validation_failed", Message: "invalid LAN DHCP settings request"},
	}}
}

func (backend *ServiceBackend) lanDHCPSettingsModule() *LanDHCPSettingsModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.gatewayPolicy == nil {
		backend.gatewayPolicy = NewDefaultGatewayPolicyModule(backend.deviceInventory)
	}
	if backend.routerContext == nil {
		backend.routerContext = NewDefaultRouterContextModule()
	}
	if backend.lanDHCPSettings == nil {
		backend.lanDHCPSettings = NewDefaultLanDHCPSettingsModule(backend.gatewayPolicy, backend.routerContext)
		backend.lanDHCPSettings.transactions = backend.taskTransactionJournalLocked()
	}
	return backend.lanDHCPSettings
}
