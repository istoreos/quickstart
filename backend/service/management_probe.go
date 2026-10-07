package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
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
	target := fmt.Sprintf("%s://%s/", request.Scheme, net.JoinHostPort(address.String(), fmt.Sprint(request.Port)))
	probeContext, cancel := context.WithTimeout(ctx, managementProbeTimeout)
	defer cancel()
	httpRequest, _ := http.NewRequestWithContext(probeContext, http.MethodHead, target, nil)
	response, requestErr := module.client.Do(httpRequest)
	result := &models.ManagementProbeResult{URL: target, DurationMS: module.now().Sub(started).Milliseconds()}
	if requestErr != nil {
		result.Error = &models.DevicePolicyError{Code: "unreachable", Message: "management interface did not respond within the safe timeout"}
		return &models.ManagementProbeResponse{Result: result}, nil
	}
	defer response.Body.Close()
	result.Reachable, result.StatusCode = true, response.StatusCode
	if module.audit != nil {
		module.audit.Record(request.DeviceID, "management_probe", "probe", "success", "")
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
	if err != nil || (!address.IsPrivate() && !address.IsLinkLocalUnicast()) {
		return netip.Addr{}, errors.New("address must be a private LAN IP")
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
				return address, nil
			}
		}
	}
	return netip.Addr{}, errors.New("address is not currently owned by this device")
}

func managementProbeFailure(code, message string) *models.ManagementProbeResponse {
	return &models.ManagementProbeResponse{Result: &models.ManagementProbeResult{Error: &models.DevicePolicyError{Code: code, Message: message}}}
}
