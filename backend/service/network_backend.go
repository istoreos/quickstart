package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/models"
)

func (backend *ServiceBackend) GetNetworkStatistic(ctx context.Context) (*models.NetworkStatisticsResponse, error) {
	return NetworkStatistic(ctx, backend.st)
}

func (backend *ServiceBackend) GetNetworkStatus(ctx context.Context, setupFinish bool) (*models.NetworkStatusResponse, error) {
	return NetworkStatus(ctx, backend.netChecker, setupFinish)
}

func (backend *ServiceBackend) GetNetworkInterfaceStatus(ctx context.Context) (*models.NetworkInterfaceStatusResponse, error) {
	return NetworkInterfaceStatus(ctx)
}

func (backend *ServiceBackend) GetNetworkInterfaceConfig(ctx context.Context) (*models.NetworkInterfaceGetConfigResponse, error) {
	return NetworkInterfaceGetConfig(ctx)
}

func (backend *ServiceBackend) SetNetworkInterfaceConfig(ctx context.Context, input NetworkInterfaceWriteInput) (*models.SDKNormalResponse, error) {
	return NetworkInterfaceSetConfig(ctx, input)
}

func (backend *ServiceBackend) PostNetworkInterfaceConfig(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return NetworkInterfacePostConfig(ctx, r)
}

func (backend *ServiceBackend) GetNetworkPortList(ctx context.Context) (*models.NetworkPortListResponse, error) {
	return NetworkPortList(ctx)
}

func (backend *ServiceBackend) GetNetworkDeviceList(ctx context.Context) (*models.DeviceListResponse, error) {
	return NetworkDeviceList(ctx)
}

func (backend *ServiceBackend) EnableNetworkHomebox(ctx context.Context) (*models.NetworkHomeBoxEnableResponse, error) {
	return NetworkHomeBoxEnable(ctx)
}

func (backend *ServiceBackend) PostNetWorkHomeboxEnable(ctx context.Context, r *http.Request) (*models.NetworkHomeBoxEnableResponse, error) {
	return backend.EnableNetworkHomebox(ctx)
}

func (backend *ServiceBackend) CheckNetworkPublicAddress(ctx context.Context, ipVersion string) (*models.NetworkCheckPublicNetResponse, error) {
	return newNetworkPublicAddressService().CheckPublicAddress(ipVersion)
}

func (backend *ServiceBackend) PostNetworkCheckPublicNet(ctx context.Context, r *http.Request) (*models.NetworkCheckPublicNetResponse, error) {
	req := models.NetworkCheckPublicNetRequest{}
	if err := getBody(&req, r); err != nil {
		return nil, errors.New("请求解析失败")
	}

	return backend.CheckNetworkPublicAddress(ctx, req.IPVersion)
}
