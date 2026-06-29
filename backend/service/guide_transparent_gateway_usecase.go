package service

import (
	"context"
	"errors"

	"github.com/linkease/quick-start/istore-backend/models"
)

type guideTransparentGatewayFacade interface {
	Get(ctx context.Context) (*models.GuideGatewayRouterRequest, error)
	Set(ctx context.Context, req models.GuideGatewayRouterRequest) (*models.SDKNormalResponse, error)
}

var newGuideTransparentGatewayServiceFacade = func() guideTransparentGatewayFacade {
	return newGuideTransparentGatewayService()
}

type GuideTransparentGatewayService struct {
	reader GuideTransparentGatewayReader
	writer GuideTransparentGatewayWriter
	apply  GuideTransparentGatewayApply
}

func newGuideTransparentGatewayService() *GuideTransparentGatewayService {
	return &GuideTransparentGatewayService{
		reader: newDefaultGuideTransparentGatewayReader(),
		writer: newDefaultGuideTransparentGatewayWriter(),
		apply:  newDefaultGuideTransparentGatewayApply(),
	}
}

func (service *GuideTransparentGatewayService) Get(ctx context.Context) (*models.GuideGatewayRouterRequest, error) {
	snapshot := service.reader.ReadTransparentGateway(ctx)
	return &models.GuideGatewayRouterRequest{
		StaticLanIP: snapshot.StaticLanIP,
		SubnetMask:  snapshot.SubnetMask,
		Gateway:     snapshot.Gateway,
		StaticDNSIP: snapshot.StaticDNSIP,
		EnableDhcp:  snapshot.EnableDhcp,
	}, nil
}

func (service *GuideTransparentGatewayService) Set(ctx context.Context, req models.GuideGatewayRouterRequest) (*models.SDKNormalResponse, error) {
	if (len(req.StaticDNSIP) == 0) || (len(req.StaticLanIP) == 0 || len(req.SubnetMask) == 0) || len(req.Gateway) == 0 {
		return nil, errors.New("missing params")
	}
	if err := service.writer.SetDHCP(ctx, req.EnableDhcp); err != nil {
		return nil, err
	}
	if err := service.writer.SetInterface(ctx, req.StaticLanIP, req.SubnetMask, req.Gateway); err != nil {
		return nil, err
	}
	if err := service.writer.SetDNS(ctx, req.StaticDNSIP); err != nil {
		return nil, err
	}
	if err := service.writer.SetLan6(ctx, req.Dhcp6c); err != nil {
		return nil, err
	}

	pending := buildGuideTransparentGatewayPending(req, service.writer.SetNat(ctx, req.EnableNat))
	if err := service.apply.Apply(ctx, pending); err != nil {
		return nil, err
	}

	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}
