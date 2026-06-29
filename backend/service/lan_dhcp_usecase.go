package service

import (
	"context"
	"errors"
	"net"
	"strings"
)

type LanDhcpService struct {
	store     DhcpConfigStore
	lanStatus LanStatusReader
}

func NewLanDhcpService(store DhcpConfigStore, lanStatus LanStatusReader) *LanDhcpService {
	return &LanDhcpService{
		store:     store,
		lanStatus: lanStatus,
	}
}

func (svc *LanDhcpService) SetDhcpGateway(ctx context.Context, input DhcpGatewayInput) error {
	lanStatus, err := svc.lanStatus.ReadLanStatus(ctx)
	if err != nil {
		return err
	}

	if input.DhcpGateway == "" {
		input.DhcpGateway = lanStatus.LanAddr
	}
	if input.DhcpGateway != "" && net.ParseIP(input.DhcpGateway) == nil {
		return errors.New("dhcp gateway is not a valid IP address")
	}

	return svc.store.ApplyGatewayConfig(ctx, input, lanStatus)
}

func (svc *LanDhcpService) SetDhcpTags(ctx context.Context, input DhcpTagConfigInput) error {
	if input.Action == "" || input.TagName == "" || input.TagTitle == "" || len(input.DhcpOption) == 0 {
		return errors.New("invalid params")
	}
	if strings.HasPrefix(input.TagName, "t_auto_") {
		return errors.New("tag name error using t_auto_")
	}

	return svc.store.ApplyTagConfig(ctx, input)
}
