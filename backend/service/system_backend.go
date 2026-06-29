package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	systemthermal "github.com/linkease/quick-start/istore-backend/modules/system/thermal"
)

func (backend *ServiceBackend) GetSystemVersion(ctx context.Context) (*models.SystemVersionResponse, error) {
	return SystemVersion(ctx)
}

func (backend *ServiceBackend) PostSystemReboot(ctx context.Context) (*models.SDKNormalResponse, error) {
	return SystemReboot(ctx)
}

func (backend *ServiceBackend) PostSystemPowerOff(ctx context.Context) (*models.SDKNormalResponse, error) {
	return SystemPowerOff(ctx)
}

func (backend *ServiceBackend) GetSystemTime(ctx context.Context) (*models.SystemTimeResponse, error) {
	return SystemTime(ctx)
}

func (backend *ServiceBackend) GetSystemCpuStatus(ctx context.Context) (*models.SystemCPUStatusResponse, error) {
	return SystemCpuStatus(ctx)
}

func (backend *ServiceBackend) GetSystemCpuTemperature(ctx context.Context) (*models.SystemCPUTemperatureResponse, error) {
	resp := models.SystemCPUTemperatureResponse{}
	resp.Result = systemthermal.BuildTemperatureResult(backend.thermalZone)
	return &resp, nil
}

func (backend *ServiceBackend) GetSystemMemoryStatus(ctx context.Context) (*models.SystemMemeryStatusResponse, error) {
	return SystemMemeryStatus(ctx)
}

func (backend *ServiceBackend) GetSystemStatus(ctx context.Context) (*models.SystemStatusResponse, error) {
	resp, err := SystemStatus(ctx)
	if err != nil {
		return nil, err
	}
	systemthermal.ApplyTemperatureToStatus(resp.Result, backend.thermalZone)
	return resp, nil
}

func (backend *ServiceBackend) GetSystemCheckUpdate(ctx context.Context) (*models.SystemCheckUpdateResponse, error) {
	return SystemCheckUpdate(ctx)
}

func (backend *ServiceBackend) PostSystemAutoCheckUpdate(ctx context.Context, req models.SystemAutoCheckUpdateRequest) (*models.SDKNormalResponse, error) {
	return SystemAutoCheckUpdateValue(ctx, req)
}

func (backend *ServiceBackend) PostSystemSetPassword(ctx context.Context, req models.NasSystemSetPasswordRequest) (*models.SDKNormalResponse, error) {
	return SystemSetPasswordValue(ctx, req)
}

func (backend *ServiceBackend) GetSystemGetSession(ctx context.Context) (*models.SystemCsrfTokenResponse, error) {
	return SystemGetSessionValue(ctx, "")
}

func (backend *ServiceBackend) PostSystemModuleSettings(ctx context.Context, req models.SystemModuleSettingsRequest) (*models.SDKNormalResponse, error) {
	return SystemModuleSettingsPostValue(ctx, req)
}

func (backend *ServiceBackend) GetSystemModuleSettings(ctx context.Context) (*models.SystemModuleSettingsResponse, error) {
	return SystemModuleSettingsGet(ctx)
}
