package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
	lancontrolspeedstats "github.com/linkease/quick-start/istore-backend/modules/lancontrol/speedstats"
)

func (backend *ServiceBackend) GetSpeedsForAllDevice(ctx context.Context, r *http.Request) (*models.DeviceSpeedStatsResponse, error) {
	lstat := backend.lstats
	hosts := lstat.reqHosts("", true)
	return lancontrolspeedstats.BuildAllDeviceResponse(lanSpeedHosts(hosts)), nil
}

func (backend *ServiceBackend) GetSpeedsForOneDevice(ctx context.Context, r *http.Request) (*models.NetworkStatisticsResponse, error) {
	var req models.SpeedsForOneDeviceRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	if req.IP == "" {
		return nil, errors.New("IP is required")
	}
	ip := req.IP
	lstat := backend.lstats
	hosts := lstat.reqHosts(ip, false)
	if len(hosts) == 0 || len(hosts[0].items) == 0 {
		return lancontrolspeedstats.BuildHistoryResponse(nil, int64(slots)), nil
	}
	return lancontrolspeedstats.BuildHistoryResponse(lanSpeedSamples(hosts[0].items), int64(slots)), nil
}

func lanSpeedHosts(hosts []*LanHostRet) []lancontrolspeedstats.Host {
	result := make([]lancontrolspeedstats.Host, 0, len(hosts))
	for _, host := range hosts {
		result = append(result, lancontrolspeedstats.Host{
			IP:      host.ip,
			Samples: lanSpeedSamples(host.items),
		})
	}
	return result
}

func lanSpeedSamples(items []*NetworkStatisticsItem) []lancontrolspeedstats.Sample {
	samples := make([]lancontrolspeedstats.Sample, 0, len(items))
	for _, item := range items {
		samples = append(samples, lancontrolspeedstats.Sample{
			StartTime:     item.startTime,
			EndTime:       item.endTime,
			UploadSpeed:   item.txAvg,
			DownloadSpeed: item.rxAvg,
		})
	}
	return samples
}

func (backend *ServiceBackend) PostLanDhcpTagsConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	var req models.LANCtrlDhcpTagConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	if err := DhcpTagsConfig(ctx, DhcpTagConfigInput{
		Action:     req.Action,
		TagName:    req.TagName,
		TagTitle:   req.TagTitle,
		DhcpOption: req.DhcpOption,
	}); err != nil {
		return nil, err
	}
	return &models.JSONResponse{}, nil
}

func (backend *ServiceBackend) PostLanDhcpGatewayConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	var req models.LANCtrlDhcpGatewayConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	if err := DhcpGatewayConfig(ctx, DhcpGatewayInput{
		DhcpEnabled: req.DhcpEnabled,
		DhcpGateway: req.DhcpGateway,
	}); err != nil {
		return nil, err
	}
	return &models.JSONResponse{}, nil
}

func (backend *ServiceBackend) PostLanSpeedLimitConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	var req models.LANCtrlSpeedLimitItem
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	if req.Action == "" {
		return nil, errors.New("action is required")
	}
	if req.Action != "delete" && req.Mac == "" {
		return nil, errors.New("mac is required")
	}
	req.Mac = strings.ToUpper(req.Mac)
	if err := newLanSpeedLimitWriteService().UpsertSpeedLimitRule(ctx, SpeedLimitWriteInput{
		Action:        req.Action,
		IP:            req.IP,
		MAC:           req.Mac,
		NetworkAccess: req.NetworkAccess,
		UploadSpeed:   req.UploadSpeed,
		DownloadSpeed: req.DownloadSpeed,
		Comment:       req.Comment,
	}); err != nil {
		return nil, err
	}
	return &models.JSONResponse{}, nil
}

func (backend *ServiceBackend) PostLanEnableSpeedLimit(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	var req models.LANCtrlSpeedLimitModule
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	if req.DownloadSpeed == 0 {
		req.DownloadSpeed = 2000
	}
	if req.UploadSpeed == 0 {
		req.UploadSpeed = 200
	}
	if err := newLanSpeedLimitWriteService().SetSpeedLimitModule(ctx, SpeedLimitModuleInput{
		Enabled:       req.Enabled,
		UploadSpeed:   req.UploadSpeed,
		DownloadSpeed: req.DownloadSpeed,
	}); err != nil {
		return nil, err
	}
	return &models.JSONResponse{}, nil
}

func (backend *ServiceBackend) PostLanEnableFloatGateway(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	var req models.LANCtrlFloatGatewayModule
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	if err := newLanFloatGatewayWriteService().SetFloatGateway(ctx, FloatGatewayWriteInput{
		Enabled:         req.Enabled,
		Role:            req.Role,
		SetIP:           req.SetIP,
		CheckIP:         req.CheckIP,
		CheckURL:        req.CheckURL,
		CheckURLTimeout: req.CheckURLTimeout,
	}); err != nil {
		return nil, err
	}
	return &models.JSONResponse{}, nil
}

func (backend *ServiceBackend) PostLanStaticDeviceConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	var req models.LANStaticAssigned
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	if err := newLanStaticAssignmentWriteService().ApplyStaticAssignment(ctx, StaticAssignmentWriteInput{
		Action:      req.Action,
		AssignedMAC: req.AssignedMac,
		AssignedIP:  req.AssignedIP,
		BindIP:      req.BindIP,
		Hostname:    req.Hostname,
		TagName:     req.TagName,
		TagTitle:    req.TagTitle,
	}); err != nil {
		return nil, err
	}
	return &models.JSONResponse{}, nil
}

func (backend *ServiceBackend) GetLanGlobalConfigs(ctx context.Context) (*models.LANCtrlGlobalConfigResponse, error) {
	return newLanGlobalConfigService().GetGlobalConfigs(ctx)
}

func (backend *ServiceBackend) GetLanListDevices(ctx context.Context) (*models.LANDeviceResponse, error) {
	return newLanDeviceListService().GetListDevices(ctx, backend)
}

func (backend *ServiceBackend) GetLanListStaticDevices(ctx context.Context) (*models.LANCtrlStaticAssignedResponse, error) {
	return newLanStaticDeviceListService().GetListStaticDevices(ctx)
}

func (backend *ServiceBackend) GetLanListSpeedLimitedDevices(ctx context.Context) (*models.LANCtrlSpeedLimitResponse, error) {
	return newLanSpeedLimitedDeviceListService().GetListSpeedLimitedDevices(ctx)
}
