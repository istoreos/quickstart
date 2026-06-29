package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/models"
)

func (backend *ServiceBackend) PostGuidePppoe(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.GuidePppoeRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return newGuidePPPoEServiceFacade().Set(ctx, req)
}

func (backend *ServiceBackend) GetGuidePppoe(ctx context.Context) (*models.GuidePppoeStatusResponse, error) {
	model, success, respErr, err := newGuidePPPoEServiceFacade().Get(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.GuidePppoeStatusResponse{Result: model, Error: respErr}
	if success != nil {
		resp.Success = *success
	}
	return &resp, nil
}

func (backend *ServiceBackend) GetGuideDockerStatus(ctx context.Context) (*models.GuideDockerStatusResponse, error) {
	return newGuideDockerRuntimeFacade().GetStatus(ctx)
}

func (backend *ServiceBackend) GetGuideDockerPartList(ctx context.Context) (*models.GuideDockerPartitionListResponse, error) {
	return newGuideDockerTransferFacade().GetPartitionList(ctx)
}

func (backend *ServiceBackend) PostGuideDockerTransfer(ctx context.Context, r *http.Request) (*models.GuideDockerTransferResponse, error) {
	req := models.GuideDockerTransferRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}

	return newGuideDockerTransferFacade().Transfer(ctx, GuideDockerTransferInput{
		Path:         req.Path,
		Force:        req.Force,
		OverwriteDir: req.OverwriteDir,
	})
}

func (backend *ServiceBackend) PostGuideDockerSwitch(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.GuideDockerSwitchRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return newGuideDockerRuntimeFacade().Switch(ctx, req.Enable)
}

func (backend *ServiceBackend) GetGuideDownloadPartList(ctx context.Context) (*models.GuideDownloadPartitionListResponse, error) {
	model, err := newGuideDownloadPartitionListServiceFacade().Get(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.GuideDownloadPartitionListResponse{Result: model}
	return &resp, nil
}

func (backend *ServiceBackend) GetGuideDownloadServiceStatus(ctx context.Context) (*models.GuideDownloadServiceResponse, error) {
	return newGuideDownloadServiceStatusFacade().Get(ctx)
}

func (backend *ServiceBackend) PostGuideQbittorrentInit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.GuideQbittorrentInitRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, errors.New("获取请求数据失败")
	}
	return newGuideQbittorrentInitServiceFacade().InitQbittorrent(ctx, GuideQbittorrentInitInput{
		ConfigPath:   req.ConfigPath,
		DownloadPath: req.DownloadPath,
	})
}

func (backend *ServiceBackend) PostGuideTransmissionInit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.GuideTransmissionInitRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return newGuideTransmissionInitServiceFacade().InitTransmission(ctx, GuideTransmissionInitInput{
		ConfigPath:   req.ConfigPath,
		DownloadPath: req.DownloadPath,
	})
}

func (backend *ServiceBackend) PostGuideAria2Init(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.GuideAria2InitRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return newGuideAria2InitServiceFacade().InitAria2(ctx, GuideAria2InitInput{
		BtTracker:    req.BtTracker,
		ConfigPath:   req.ConfigPath,
		DownloadPath: req.DownloadPath,
		RPCToken:     req.RPCToken,
	})
}

func (backend *ServiceBackend) PostGuideLan(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.GuideLanSettingRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return newGuideLanSettingServiceFacade().Set(ctx, req)
}

func (backend *ServiceBackend) GetGuideLan(ctx context.Context) (*models.GuideLanSettingResponse, error) {
	model, err := newGuideLanSettingServiceFacade().Get(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.GuideLanSettingResponse{Result: model}
	return &resp, nil
}

func (backend *ServiceBackend) PostGuideClientMode(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.GuideClientModeRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return newGuideDhcpClientServiceFacade().Set(ctx, req)
}

func (backend *ServiceBackend) GetGuideClientMode(ctx context.Context) (*models.GuideClientModeResponse, error) {
	model, success, respErr, err := newGuideDhcpClientServiceFacade().Get(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.GuideClientModeResponse{Result: model, Error: respErr}
	if success != nil {
		resp.Success = *success
	}
	return &resp, nil
}

func (backend *ServiceBackend) PostGuideGatewayRouter(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.GuideGatewayRouterRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return newGuideTransparentGatewayServiceFacade().Set(ctx, req)
}

func (backend *ServiceBackend) PostGuideDnsConfig(ctx context.Context, r *http.Request) (*models.GuideDNSConfigResponse, error) {
	req := models.GuideDNSConfigRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	model, err := newGuideDNSConfigServiceFacade().Set(ctx, req)
	if err != nil {
		return nil, err
	}
	resp := models.GuideDNSConfigResponse{Result: model}
	return &resp, nil
}

func (backend *ServiceBackend) GetGuideDnsConfig(ctx context.Context) (*models.GuideDNSConfigResponse, error) {
	model, err := newGuideDNSConfigServiceFacade().Get(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.GuideDNSConfigResponse{Result: model}
	return &resp, nil
}

func (backend *ServiceBackend) PostGuideSoftSource(ctx context.Context, r *http.Request) (*models.GuideSoftSourceResponse, error) {
	req := models.GuideSoftSourceRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, errors.New("请求解析失败")
	}
	model, err := guideSoftSourceSet(ctx, GuideSoftSourceInput{
		SoftSourceIdentity: req.SoftSourceIdentity,
	})
	if err != nil {
		return nil, err
	}
	resp := models.GuideSoftSourceResponse{Result: model}
	return &resp, nil
}

func (backend *ServiceBackend) GetGuideSoftSource(ctx context.Context) (*models.GuideSoftSourceResponse, error) {
	model, err := guideSoftSourceGet(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.GuideSoftSourceResponse{Result: model}
	return &resp, nil
}

func (backend *ServiceBackend) GetGuideSoftSourceList(ctx context.Context) (*models.GuideSoftSourceListResponse, error) {
	model, err := guideSoftSourceList(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.GuideSoftSourceListResponse{Result: model}
	return &resp, nil
}

func (backend *ServiceBackend) GetGuideDdns(ctx context.Context) (*models.GuideDdnsResponse, error) {
	model, err := newGuideDDNSStatusServiceFacade().Get(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.GuideDdnsResponse{Result: model}
	return &resp, nil
}

func (backend *ServiceBackend) PostGuideDdns(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.GuideDdnsRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, errors.New("请求解析失败")
	}
	var sessionID string
	if sessionCookie, _ := r.Cookie("sysauth"); sessionCookie != nil {
		sessionID = sessionCookie.Value
	}
	return guideDDNSUpdate(ctx, GuideDDNSInput{
		SessionID:   sessionID,
		Domain:      req.Domain,
		IPVersion:   req.IPVersion,
		Password:    req.Password,
		ServiceName: req.ServiceName,
		UserName:    req.UserName,
	})
}

func (backend *ServiceBackend) PostGuideDdnsto(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.GuideDdnstoRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, errors.New("请求解析失败")
	}
	return guideDdnstoEnable(ctx, GuideDdnstoEnableInput{
		Token: req.Token,
	})
}

func (backend *ServiceBackend) GetGuideDdnstoConfig(ctx context.Context) (*models.GuideDdnstoConfigResponse, error) {
	model, err := newGuideDdnstoConfigServiceFacade().Get(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.GuideDdnstoConfigResponse{Result: model}
	return &resp, nil
}

func (backend *ServiceBackend) PostGuideDdnstoAddress(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.GuideDdnstoAddressRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, errors.New("请求解析失败")
	}
	return guideDdnstoAddress(ctx, GuideDdnstoAddressInput{
		Address: req.Address,
	})
}
