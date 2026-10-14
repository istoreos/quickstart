package service

import (
	"context"
	"errors"
	"net"
	"net/netip"
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
	if input.Action != "add" && input.Action != "modify" && input.Action != "delete" {
		return errors.New("invalid DHCP tag action")
	}
	if input.TagName == "" {
		return errors.New("invalid params")
	}
	if !dhcpTagNamePattern.MatchString(input.TagName) {
		return errors.New("invalid DHCP tag identifier")
	}
	if strings.HasPrefix(input.TagName, "t_auto_") {
		return errors.New("tag name error using t_auto_")
	}
	if input.Action == "delete" {
		input.TagTitle = ""
		input.DhcpOption = nil
		return svc.store.ApplyTagConfig(ctx, input)
	}
	if input.TagTitle == "" || len(input.DhcpOption) == 0 {
		return errors.New("invalid params")
	}
	if !validSafeLabel(input.TagTitle, 128) {
		return errors.New("invalid DHCP tag title")
	}
	for _, option := range input.DhcpOption {
		parts := strings.Split(option, ",")
		if len(parts) != 2 || (parts[0] != "3" && parts[0] != "6") {
			return errors.New("DHCP tag options must contain only IPv4 gateway or DNS server addresses")
		}
		address, err := netip.ParseAddr(parts[1])
		if err != nil || !address.Is4() {
			return errors.New("DHCP tag options must contain only IPv4 gateway or DNS server addresses")
		}
	}

	return svc.store.ApplyTagConfig(ctx, input)
}
