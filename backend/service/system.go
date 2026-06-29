package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/models"
)

func SystemVersion(ctx context.Context) (*models.SystemVersionResponse, error) {
	result, err := newSystemVersionService().Get(ctx)
	if err != nil {
		return nil, err
	}

	resp := models.SystemVersionResponse{}
	resp.Result = result
	return &resp, nil
}

func SystemTime(ctx context.Context) (*models.SystemTimeResponse, error) {
	result, err := newSystemRuntimeService().Time(ctx)
	if err != nil {
		return nil, err
	}

	resp := models.SystemTimeResponse{}
	resp.Result = result
	return &resp, nil
}

func SystemStatus(ctx context.Context) (*models.SystemStatusResponse, error) {
	result, err := newSystemRuntimeService().Status(ctx)
	if err != nil {
		return nil, err
	}

	resp := models.SystemStatusResponse{}
	resp.Result = result
	return &resp, nil
}

func SystemCpuStatus(ctx context.Context) (*models.SystemCPUStatusResponse, error) {
	result, err := newSystemRuntimeService().CPU(ctx)
	if err != nil {
		return nil, err
	}

	resp := models.SystemCPUStatusResponse{}
	resp.Result = result
	return &resp, nil
}

func SystemMemeryStatus(ctx context.Context) (*models.SystemMemeryStatusResponse, error) {
	result, err := newSystemRuntimeService().Memory(ctx)
	if err != nil {
		return nil, err
	}

	resp := models.SystemMemeryStatusResponse{}
	resp.Result = result
	return &resp, nil
}

func SystemCheckUpdate(ctx context.Context) (*models.SystemCheckUpdateResponse, error) {
	result, err := newSystemUpdateService().Check(ctx)
	if err != nil {
		return nil, err
	}

	resp := models.SystemCheckUpdateResponse{}
	resp.Result = result
	return &resp, nil
}

func SystemAutoCheckUpdate(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.SystemAutoCheckUpdateRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, errors.New("请求解析失败")
	}

	return SystemAutoCheckUpdateValue(ctx, req)
}

func SystemAutoCheckUpdateValue(ctx context.Context, req models.SystemAutoCheckUpdateRequest) (*models.SDKNormalResponse, error) {
	return newSystemUpdateService().SetAutoCheck(ctx, req)
}

func SystemReboot(ctx context.Context) (*models.SDKNormalResponse, error) {
	return newSystemPowerService().Reboot(ctx)
}

func SystemPowerOff(ctx context.Context) (*models.SDKNormalResponse, error) {
	return newSystemPowerService().PowerOff(ctx)
}

func SystemSetPassword(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.NasSystemSetPasswordRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, errors.New("请求解析失败")
	}

	return SystemSetPasswordValue(ctx, req)
}

func SystemSetPasswordValue(ctx context.Context, req models.NasSystemSetPasswordRequest) (*models.SDKNormalResponse, error) {
	return newSystemPasswordService().SetRootPassword(ctx, req)
}

func SystemGetSession(ctx context.Context, r *http.Request) (*models.SystemCsrfTokenResponse, error) {
	return SystemGetSessionValue(ctx, systemSessionIDFromRequest(r))
}

func SystemGetSessionValue(ctx context.Context, sessionID string) (*models.SystemCsrfTokenResponse, error) {
	if sessionID == "" {
		sessionID = systemSessionIDFromContext(ctx)
	}
	result, err := newSystemSessionService().Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	resp := models.SystemCsrfTokenResponse{Result: result}
	return &resp, nil
}

const systemSessionIDContextKey = "github.com/linkease/quick-start/istore-backend/system/session-id"

func systemSessionIDFromContext(ctx context.Context) string {
	sessionID, _ := ctx.Value(systemSessionIDContextKey).(string)
	return sessionID
}

func SystemModuleSettingsPost(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	var req models.SystemModuleSettingsRequest
	err := getBody(&req, r)
	if err != nil {
		return nil, errors.New("请求解析失败")
	}

	return SystemModuleSettingsPostValue(ctx, req)
}

func SystemModuleSettingsPostValue(ctx context.Context, req models.SystemModuleSettingsRequest) (*models.SDKNormalResponse, error) {
	return newSystemModuleSettingsService().Set(ctx, req)
}

func SystemModuleSettingsGet(ctx context.Context) (*models.SystemModuleSettingsResponse, error) {
	result, err := newSystemModuleSettingsService().Get(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.SystemModuleSettingsResponse{Result: result}
	return &resp, nil
}
