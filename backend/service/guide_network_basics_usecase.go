package service

import (
	"context"
	"errors"
	"net"

	"github.com/linkease/quick-start/istore-backend/models"
)

type guideDNSConfigFacade interface {
	Get(ctx context.Context) (*models.GuideDNSConfigResponseResult, error)
	Set(ctx context.Context, req models.GuideDNSConfigRequest) (*models.GuideDNSConfigResponseResult, error)
}

var newGuideDNSConfigServiceFacade = func() guideDNSConfigFacade {
	return newGuideDNSConfigService()
}

type GuideDNSConfigService struct {
	reader GuideNetworkBasicsReader
	writer GuideNetworkBasicsWriter
	apply  GuideNetworkBasicsApply
}

func newGuideDNSConfigService() *GuideDNSConfigService {
	return &GuideDNSConfigService{
		reader: newDefaultGuideNetworkBasicsReader(),
		writer: newDefaultGuideNetworkBasicsWriter(),
		apply:  newDefaultGuideNetworkBasicsApply(),
	}
}

func (service *GuideDNSConfigService) Get(ctx context.Context) (*models.GuideDNSConfigResponseResult, error) {
	snapshot, err := service.reader.ReadDNSConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &models.GuideDNSConfigResponseResult{
		DNSProto:      snapshot.DNSProto,
		InterfaceName: snapshot.InterfaceName,
		ManualDNSIP:   snapshot.ManualDNSIP,
	}, nil
}

func (service *GuideDNSConfigService) Set(ctx context.Context, req models.GuideDNSConfigRequest) (*models.GuideDNSConfigResponseResult, error) {
	if req.DNSProto == "" || (req.DNSProto == "manual" && len(req.ManualDNSIP) == 0) || (req.DNSProto != "manual" && req.DNSProto != "auto") {
		return nil, errors.New("missing params")
	}

	defaultIf, err := service.reader.ReadDefaultOutboundInterface(ctx)
	if err != nil {
		return nil, err
	}

	if req.DNSProto != "manual" && defaultIf.Proto == "static" {
		return nil, errors.New("dns must be set when using static proto")
	}

	if err := service.writer.SetDNSConfig(ctx, GuideSetDNSConfigInput{
		InterfaceName: defaultIf.InterfaceName,
		DNSProto:      req.DNSProto,
		ManualDNSIP:   req.ManualDNSIP,
	}); err != nil {
		return nil, err
	}

	if err := service.apply.Apply(ctx, []string{"network"}); err != nil {
		return nil, err
	}

	result := &models.GuideDNSConfigResponseResult{
		DNSProto:      req.DNSProto,
		InterfaceName: defaultIf.InterfaceName,
	}
	if req.DNSProto != "auto" {
		result.ManualDNSIP = req.ManualDNSIP
	}
	return result, nil
}

type guideDhcpClientFacade interface {
	Get(ctx context.Context) (*models.GuideClientModeResponseResult, *models.ResponseSuccess, models.ResponseError, error)
	Set(ctx context.Context, req models.GuideClientModeRequest) (*models.SDKNormalResponse, error)
}

var newGuideDhcpClientServiceFacade = func() guideDhcpClientFacade {
	return newGuideDhcpClientService()
}

type GuideDhcpClientService struct {
	reader GuideNetworkBasicsReader
	writer GuideNetworkBasicsWriter
	apply  GuideNetworkBasicsApply
}

func newGuideDhcpClientService() *GuideDhcpClientService {
	return &GuideDhcpClientService{
		reader: newDefaultGuideNetworkBasicsReader(),
		writer: newDefaultGuideNetworkBasicsWriter(),
		apply:  newDefaultGuideNetworkBasicsApply(),
	}
}

func (service *GuideDhcpClientService) Get(ctx context.Context) (*models.GuideClientModeResponseResult, *models.ResponseSuccess, models.ResponseError, error) {
	wan := service.reader.ReadWANConfig(ctx)
	if wan == nil || !wan.Exists {
		success := models.ResponseSuccess(int64(NetworkErrorWanNotExists))
		err := models.ResponseError(NetworkErrorMessageWanNotExists)
		return nil, &success, err, nil
	}

	result := &models.GuideClientModeResponseResult{
		WanProto:    wan.WanProto,
		StaticIP:    wan.StaticIP,
		SubnetMask:  wan.SubnetMask,
		Gateway:     wan.Gateway,
		DNSProto:    wan.DNSProto,
		ManualDNSIP: wan.ManualDNSIP,
	}

	if result.StaticIP == "" && result.WanProto == "dhcp" {
		runtime, err := service.reader.ReadWANRuntime(ctx, "wan")
		if err == nil && runtime != nil {
			result.StaticIP = runtime.StaticIP
			result.SubnetMask = runtime.SubnetMask
			if result.Gateway == "" {
				result.Gateway = runtime.Gateway
			}
		}
	}
	return result, nil, "", nil
}

func (service *GuideDhcpClientService) Set(ctx context.Context, req models.GuideClientModeRequest) (*models.SDKNormalResponse, error) {
	if req.WanProto != "static" && req.WanProto != "dhcp" {
		return nil, errors.New("WanProto should be static or dhcp")
	}
	if req.WanProto == "static" && (len(req.Gateway) == 0 || len(req.SubnetMask) == 0) {
		return nil, errors.New("gateway or netmask missing")
	}
	if req.WanProto == "static" && req.DNSProto == "auto" {
		success := models.ResponseSuccess(int64(NetworkErrorDnsNotSetting))
		err := models.ResponseError("静态IP地址，dns必须手动配置")
		return &models.SDKNormalResponse{Error: err, Success: &success}, nil
	}
	if req.DNSProto == "manual" && len(req.ManualDNSIP) == 0 {
		return nil, errors.New(" DNS IP missing ")
	}
	if req.DNSProto != "manual" && req.DNSProto != "auto" {
		return nil, errors.New("incorrect DNSProto")
	}

	if err := service.writer.SetWANInterfaceMode(ctx, GuideSetWANInterfaceInput{
		InterfaceName: "wan",
		WanProto:      req.WanProto,
		StaticIP:      req.StaticIP,
		SubnetMask:    req.SubnetMask,
		Gateway:       req.Gateway,
	}); err != nil {
		return nil, err
	}
	if err := service.writer.SetDNSConfig(ctx, GuideSetDNSConfigInput{
		InterfaceName: "wan",
		DNSProto:      req.DNSProto,
		ManualDNSIP:   req.ManualDNSIP,
	}); err != nil {
		return nil, err
	}
	if err := writeGuideNetworkBasicsDeleteLan6(ctx); err != nil {
		return nil, err
	}

	pending := buildGuideNetworkBasicsPendingForWANMode(req, writeGuideNetworkBasicsSetLanMasq(ctx, false))
	if req.EnableLanDhcp {
		writeGuideNetworkBasicsEnableLanDHCP(ctx)
	}
	if err := service.apply.Apply(ctx, pending); err != nil {
		return nil, err
	}
	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}

type guidePPPoEFacade interface {
	Get(ctx context.Context) (*models.GuidePppoeStatusResponseResult, *models.ResponseSuccess, models.ResponseError, error)
	Set(ctx context.Context, req models.GuidePppoeRequest) (*models.SDKNormalResponse, error)
}

var newGuidePPPoEServiceFacade = func() guidePPPoEFacade {
	return newGuidePPPoEService()
}

type GuidePPPoEService struct {
	reader GuideNetworkBasicsReader
	writer GuideNetworkBasicsWriter
	apply  GuideNetworkBasicsApply
}

func newGuidePPPoEService() *GuidePPPoEService {
	return &GuidePPPoEService{
		reader: newDefaultGuideNetworkBasicsReader(),
		writer: newDefaultGuideNetworkBasicsWriter(),
		apply:  newDefaultGuideNetworkBasicsApply(),
	}
}

func (service *GuidePPPoEService) Get(ctx context.Context) (*models.GuidePppoeStatusResponseResult, *models.ResponseSuccess, models.ResponseError, error) {
	wan := service.reader.ReadWANConfig(ctx)
	if wan == nil || !wan.Exists {
		success := models.ResponseSuccess(int64(NetworkErrorWanNotExists))
		err := models.ResponseError(NetworkErrorMessageWanNotExists)
		return nil, &success, err, nil
	}
	return &models.GuidePppoeStatusResponseResult{
		Account:  wan.PPPoEAccount,
		Password: wan.PPPoEPassword,
	}, nil, "", nil
}

func (service *GuidePPPoEService) Set(ctx context.Context, req models.GuidePppoeRequest) (*models.SDKNormalResponse, error) {
	if len(req.Account) == 0 || len(req.Password) == 0 {
		return nil, errors.New("missing params")
	}
	if err := service.writer.SetPPPoE(ctx, GuideSetPPPoEInput{
		Account:  req.Account,
		Password: req.Password,
	}); err != nil {
		return nil, err
	}
	if err := writeGuideNetworkBasicsDeleteLan6(ctx); err != nil {
		return nil, err
	}
	pending := buildGuideNetworkBasicsPendingForWANMode(req, writeGuideNetworkBasicsSetLanMasq(ctx, false))
	if req.EnableLanDhcp {
		writeGuideNetworkBasicsEnableLanDHCP(ctx)
	}
	if err := service.apply.Apply(ctx, pending); err != nil {
		return nil, err
	}
	return &models.SDKNormalResponse{}, nil
}

type guideLanSettingFacade interface {
	Get(ctx context.Context) (*models.GuideLanSettingResponseResult, error)
	Set(ctx context.Context, req models.GuideLanSettingRequest) (*models.SDKNormalResponse, error)
}

var newGuideLanSettingServiceFacade = func() guideLanSettingFacade {
	return newGuideLanSettingService()
}

type GuideLanSettingService struct {
	reader GuideNetworkBasicsReader
	writer GuideNetworkBasicsWriter
	apply  GuideNetworkBasicsApply
}

func newGuideLanSettingService() *GuideLanSettingService {
	return &GuideLanSettingService{
		reader: newDefaultGuideNetworkBasicsReader(),
		writer: newDefaultGuideNetworkBasicsWriter(),
		apply:  newDefaultGuideNetworkBasicsApply(),
	}
}

func (service *GuideLanSettingService) Get(ctx context.Context) (*models.GuideLanSettingResponseResult, error) {
	lan := service.reader.ReadLANConfig(ctx)
	return &models.GuideLanSettingResponseResult{
		LanIP:      lan.LanIP,
		NetMask:    lan.NetMask,
		EnableDhcp: lan.EnableDhcp,
		DhcpStart:  lan.DhcpStart,
		DhcpEnd:    lan.DhcpEnd,
	}, nil
}

func (service *GuideLanSettingService) Set(ctx context.Context, req models.GuideLanSettingRequest) (*models.SDKNormalResponse, error) {
	if len(req.LanIP) == 0 || len(req.NetMask) == 0 || (req.EnableDhcp && (len(req.DhcpStart) == 0 || len(req.DhcpEnd) == 0)) {
		return nil, errors.New("missing params")
	}
	if req.EnableDhcp && (net.ParseIP(req.DhcpStart) == nil || net.ParseIP(req.DhcpEnd) == nil) {
		return nil, errors.New("IP池起始或结束地址错误")
	}

	pending, err := service.writer.SetLANConfig(ctx, GuideSetLANConfigInput{
		LanIP:      req.LanIP,
		NetMask:    req.NetMask,
		EnableDhcp: req.EnableDhcp,
		DhcpStart:  req.DhcpStart,
		DhcpEnd:    req.DhcpEnd,
	})
	if err != nil {
		return nil, err
	}

	applyPending := make([]string, 0, len(pending))
	for _, item := range pending {
		if item == "network" {
			continue
		}
		applyPending = append(applyPending, item)
	}
	if len(applyPending) > 0 {
		if err := service.apply.Apply(ctx, applyPending); err != nil {
			return nil, err
		}
	}

	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}
