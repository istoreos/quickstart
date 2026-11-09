package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

const managementProbeTimeout = 2 * time.Second

type ManagementProbeModule struct {
	inventory  *DeviceInventoryModule
	audit      *NetworkAuditModule
	probeSlots chan struct{}
	now        func() time.Time
	client     *http.Client
}

func NewManagementProbeModule(inventory *DeviceInventoryModule, audit *NetworkAuditModule) *ManagementProbeModule {
	transport := &http.Transport{
		Proxy:             nil,
		DialContext:       (&net.Dialer{Timeout: managementProbeTimeout}).DialContext,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, // LAN devices commonly use self-signed certificates; response content is never read.
		DisableKeepAlives: true,
	}
	return &ManagementProbeModule{
		inventory: inventory, audit: audit, probeSlots: make(chan struct{}, 4), now: time.Now,
		client: &http.Client{Transport: transport, Timeout: managementProbeTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
}

func (module *AdvancedNetworkModule) Probe(ctx context.Context, request *models.ManagementProbeRequest) (*models.ManagementProbeResponse, error) {
	return module.management.Probe(ctx, request)
}

func (module *ManagementProbeModule) Probe(ctx context.Context, request *models.ManagementProbeRequest) (*models.ManagementProbeResponse, error) {
	started := module.now()
	address, err := module.validateTarget(ctx, request)
	if err != nil {
		return managementProbeFailure("validation_failed", err.Error()), nil
	}
	select {
	case module.probeSlots <- struct{}{}:
		defer func() { <-module.probeSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	scopedClient, err := module.clientForScope(ctx, address)
	if err != nil {
		return managementProbeFailure("network_source_unavailable", err.Error()), nil
	}
	target := (&url.URL{Scheme: request.Scheme, Host: net.JoinHostPort(address.String(), fmt.Sprint(request.Port)), Path: "/"}).String()
	probeContext, cancel := context.WithTimeout(ctx, managementProbeTimeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(probeContext, http.MethodHead, target, nil)
	if err != nil {
		return managementProbeFailure("validation_failed", "无效的设备管理地址"), nil
	}
	response, requestErr := scopedClient.Do(httpRequest)
	result := &models.ManagementProbeResult{URL: target, DurationMS: module.now().Sub(started).Milliseconds()}
	if requestErr != nil {
		result.Error = &models.DevicePolicyError{Code: "unreachable", Message: "management interface did not respond within the safe timeout"}
		return &models.ManagementProbeResponse{Result: result}, nil
	}
	defer response.Body.Close()
	result.Reachable, result.StatusCode = true, response.StatusCode
	if module.audit != nil {
		module.audit.forContext(ctx).Record(request.DeviceID, "management_probe", "probe", "success", "")
	}
	return &models.ManagementProbeResponse{Result: result}, nil
}

func (module *ManagementProbeModule) validateTarget(ctx context.Context, request *models.ManagementProbeRequest) (netip.Addr, error) {
	if request == nil || strings.TrimSpace(request.DeviceID) == "" {
		return netip.Addr{}, errors.New("deviceId is required")
	}
	if request.Scheme != "http" && request.Scheme != "https" {
		return netip.Addr{}, errors.New("scheme must be http or https")
	}
	allowedPorts := map[int]bool{80: true, 443: true, 8080: true, 8443: true}
	if !allowedPorts[request.Port] {
		return netip.Addr{}, errors.New("port is not in the management probe allowlist")
	}
	address, err := netip.ParseAddr(strings.TrimSpace(request.Address))
	if err != nil || !address.IsValid() || address.IsUnspecified() || address.IsMulticast() || address.IsLoopback() {
		return netip.Addr{}, errors.New("address must be a LAN IP")
	}
	scope := selectedLANScope(ctx)
	allowed := address.IsPrivate() || address.IsLinkLocalUnicast()
	if scope != nil {
		if address.Zone() != "" && address.Zone() != scope.Device {
			return netip.Addr{}, errors.New("address belongs to another link")
		}
		for _, cidr := range append(append([]string{}, scope.IPv4Prefixes...), scope.IPv6Prefixes...) {
			if prefix, err := netip.ParsePrefix(cidr); err == nil && prefix.Contains(address.WithZone("")) {
				allowed = true
			}
		}
	}
	if !allowed {
		return netip.Addr{}, errors.New("address must be a LAN IP")
	}

	if module.inventory == nil {
		return netip.Addr{}, errors.New("device inventory is unavailable")
	}
	inventory, err := module.inventory.Snapshot(ctx)
	if err != nil {
		return netip.Addr{}, err
	}
	for _, device := range inventory.Result.Devices {
		if device == nil || device.DeviceID != request.DeviceID || device.Addresses == nil {
			continue
		}
		for _, candidate := range device.Addresses.Current {
			if candidate != nil && canonicalInventoryAddress(candidate.Address) == canonicalInventoryAddress(address.String()) {
				if address.Is6() && address.IsLinkLocalUnicast() {
					scope := selectedLANScope(ctx)
					if scope == nil || scope.Device == "" {
						return netip.Addr{}, errors.New("link-local address requires its network")
					}
					address = address.WithZone(scope.Device)
				}
				return address, nil
			}
		}
	}
	return netip.Addr{}, errors.New("address is not currently owned by this device")
}

func managementProbeFailure(code, message string) *models.ManagementProbeResponse {
	return &models.ManagementProbeResponse{Result: &models.ManagementProbeResult{Error: &models.DevicePolicyError{Code: code, Message: message}}}
}

func (module *ManagementProbeModule) clientForScope(ctx context.Context, address netip.Addr) (*http.Client, error) {
	scope := selectedLANScope(ctx)
	if scope == nil {
		return module.client, nil
	}
	var source net.IP
	prefixes := scope.IPv4Prefixes
	if address.Is6() {
		prefixes = scope.IPv6Prefixes
	}
	// Use an address actually assigned to this device, never a subnet's network
	// address or the browser-provided target as the source address.
	iface, err := net.InterfaceByName(scope.Device)
	if err != nil {
		return nil, err
	}
	assigned, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	for _, value := range assigned {
		prefix, err := netip.ParsePrefix(value.String())
		if err != nil || prefix.Addr().Is6() != address.Is6() {
			continue
		}
		for _, allowed := range prefixes {
			network, err := netip.ParsePrefix(allowed)
			if err == nil && network.Contains(prefix.Addr()) {
				source = net.IP(prefix.Addr().AsSlice())
				break
			}
		}
		if source != nil {
			break
		}
	}
	if source == nil {
		return nil, errors.New("当前网络没有可用的探测源地址")
	}
	transport, ok := module.client.Transport.(*http.Transport)
	if !ok {
		return nil, errors.New("network-bound probe transport unavailable")
	}
	cloned := transport.Clone()
	dialer := &net.Dialer{Timeout: managementProbeTimeout, LocalAddr: &net.TCPAddr{IP: source, Zone: scope.Device}, Control: func(_, _ string, raw syscall.RawConn) error {
		var bindErr error
		if err := raw.Control(func(fd uintptr) {
			bindErr = unix.SetsockoptString(int(fd), unix.SOL_SOCKET, unix.SO_BINDTODEVICE, scope.Device)
		}); err != nil {
			return err
		}
		return bindErr
	}}
	cloned.DialContext = dialer.DialContext
	client := *module.client
	client.Transport = cloned
	return &client, nil
}
