package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

func GuideGetLanSetting(ctx context.Context) (*models.GuideLanSettingResponse, error) {
	model, err := newGuideLanSettingServiceFacade().Get(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.GuideLanSettingResponse{Result: model}
	return &resp, nil
}

func SetTransparentGateway(ctx context.Context, req *models.GuideGatewayRouterRequest) (*models.SDKNormalResponse, error) {
	return newGuideTransparentGatewayServiceFacade().Set(ctx, *req)
}

func GuideGetTransparentGateway() (*models.GuideGatewayRouterRequest, error) {
	return newGuideTransparentGatewayServiceFacade().Get(context.Background())
}
