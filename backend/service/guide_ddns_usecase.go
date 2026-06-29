package service

import (
	"context"
	"errors"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/guideddns"
)

type GuideDdnstoEnableService struct {
	writer GuideDDNSWriter
}

type GuideDdnstoAddressService struct {
	writer GuideDDNSWriter
}

type GuideDDNSInput struct {
	SessionID   string
	Domain      string
	IPVersion   string
	Password    string
	ServiceName string
	UserName    string
}

type GuideDDNSService struct {
	reader GuideDDNSReader
	writer GuideDDNSWriter
}

var newGuideDdnstoEnableFacade = func() *GuideDdnstoEnableService {
	return newGuideDdnstoEnableService()
}

var newGuideDdnstoAddressFacade = func() *GuideDdnstoAddressService {
	return newGuideDdnstoAddressService()
}

var newGuideDDNSFacade = func() *GuideDDNSService {
	return newGuideDDNSService()
}

var guideDdnstoEnable = func(ctx context.Context, input GuideDdnstoEnableInput) (*models.SDKNormalResponse, error) {
	return newGuideDdnstoEnableFacade().Enable(ctx, input)
}

var guideDdnstoAddress = func(ctx context.Context, input GuideDdnstoAddressInput) (*models.SDKNormalResponse, error) {
	return newGuideDdnstoAddressFacade().UpdateAddress(ctx, input)
}

var guideDDNSUpdate = func(ctx context.Context, input GuideDDNSInput) (*models.SDKNormalResponse, error) {
	return newGuideDDNSFacade().Update(ctx, input)
}

func newGuideDdnstoEnableService() *GuideDdnstoEnableService {
	return &GuideDdnstoEnableService{
		writer: newDefaultGuideDDNSWriter(),
	}
}

func newGuideDdnstoAddressService() *GuideDdnstoAddressService {
	return &GuideDdnstoAddressService{
		writer: newDefaultGuideDDNSWriter(),
	}
}

func newGuideDDNSService() *GuideDDNSService {
	return &GuideDDNSService{
		reader: newDefaultGuideDDNSReader(),
		writer: newDefaultGuideDDNSWriter(),
	}
}

func (service *GuideDdnstoEnableService) Enable(ctx context.Context, input GuideDdnstoEnableInput) (*models.SDKNormalResponse, error) {
	stderr, err := service.writer.EnableDdnsto(ctx, input)
	if err != nil {
		return nil, errors.New("ddnsto启动失败" + stderr)
	}
	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (service *GuideDdnstoAddressService) UpdateAddress(ctx context.Context, input GuideDdnstoAddressInput) (*models.SDKNormalResponse, error) {
	stderr, err := service.writer.UpdateDdnstoAddress(ctx, input)
	if err != nil {
		return nil, errors.New("ddnsto地址信息保存失败" + stderr)
	}
	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (service *GuideDDNSService) Update(ctx context.Context, input GuideDDNSInput) (*models.SDKNormalResponse, error) {
	if input.SessionID != "" {
		pending, err := service.reader.ReadDDNSPendingChanges(ctx, input.SessionID)
		if err != nil {
			return nil, errors.New("获取ddns编辑状态失败")
		}
		if pending {
			return &models.SDKNormalResponse{
				Error: models.ResponseError("-100"),
				Scope: models.ResponseScope("guide.ddns"),
			}, nil
		}
	}

	outbound, err := service.reader.ReadOutboundInterfaces(ctx)
	if err != nil {
		return nil, err
	}

	runtime, err := service.resolveRuntime(input.IPVersion, outbound)
	if err != nil {
		return nil, err
	}
	serviceName, err := guideddns.BuildGuideDDNSServiceName(input.ServiceName)
	if err != nil {
		return nil, err
	}

	cmds := guideddns.BuildGuideDDNSApplyCommands(guideddns.GuideDDNSApplyCommandInput{
		ConfigName:  runtime.ConfigName,
		UseIPv6:     runtime.UseIPv6,
		ServiceName: serviceName,
		Domain:      input.Domain,
		UserName:    strings.TrimSpace(input.UserName),
		Password:    strings.TrimSpace(input.Password),
		Interface:   runtime.Interface,
		HasPublic:   runtime.HasPublic,
		IPURL:       runtime.IPURL,
	})
	if err := service.writer.ApplyDDNSConfig(ctx, cmds); err != nil {
		return nil, errors.New("修改ddns配置失败")
	}

	// Legacy behavior ignores start failures after the config has been committed.
	_ = service.writer.StartDDNSService(ctx, runtime.ConfigName)

	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (service *GuideDDNSService) resolveRuntime(ipVersion string, outbound *GuideDDNSOutboundSnapshot) (guideddns.GuideDDNSRuntimeResolution, error) {
	snapshot := guideddns.GuideDDNSRuntimeSnapshot{}
	if outbound != nil && outbound.IPv4 != nil {
		snapshot.IPv4 = &guideddns.GuideDDNSRuntimeInterfaceSnapshot{
			InterfaceName: outbound.IPv4.InterfaceName,
			IP:            outbound.IPv4.IP,
			Public:        service.reader.IsPublicIPv4(outbound.IPv4.IP),
		}
	}
	if outbound != nil && outbound.IPv6 != nil {
		snapshot.IPv6 = &guideddns.GuideDDNSRuntimeInterfaceSnapshot{
			InterfaceName: outbound.IPv6.InterfaceName,
			IP:            outbound.IPv6.IP,
			Public:        service.reader.IsPublicIPv6(outbound.IPv6.IP),
		}
	}
	return guideddns.ResolveGuideDDNSRuntime(ipVersion, snapshot)
}
