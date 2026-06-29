package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeGuideTransparentGatewayReader struct {
	snapshot *GuideTransparentGatewaySnapshot
}

func (reader *fakeGuideTransparentGatewayReader) ReadTransparentGateway(ctx context.Context) *GuideTransparentGatewaySnapshot {
	return reader.snapshot
}

type fakeGuideTransparentGatewayWriter struct {
	dhcpEnabled *bool
	dhcpErr     error

	interfaceStaticIP string
	interfaceMask     string
	interfaceGateway  string
	interfaceErr      error

	dnsIP  string
	dnsErr error

	lan6Enabled *bool
	lan6Err     error

	natEnabled *bool
	natChanged bool
}

func (writer *fakeGuideTransparentGatewayWriter) SetDHCP(ctx context.Context, enable bool) error {
	writer.dhcpEnabled = &enable
	return writer.dhcpErr
}

func (writer *fakeGuideTransparentGatewayWriter) SetInterface(ctx context.Context, staticIP string, subnetMask string, gateway string) error {
	writer.interfaceStaticIP = staticIP
	writer.interfaceMask = subnetMask
	writer.interfaceGateway = gateway
	return writer.interfaceErr
}

func (writer *fakeGuideTransparentGatewayWriter) SetDNS(ctx context.Context, dnsIP string) error {
	writer.dnsIP = dnsIP
	return writer.dnsErr
}

func (writer *fakeGuideTransparentGatewayWriter) SetLan6(ctx context.Context, enable bool) error {
	writer.lan6Enabled = &enable
	return writer.lan6Err
}

func (writer *fakeGuideTransparentGatewayWriter) SetNat(ctx context.Context, enable bool) bool {
	writer.natEnabled = &enable
	return writer.natChanged
}

type fakeGuideTransparentGatewayApply struct {
	pending []string
	err     error
}

func (apply *fakeGuideTransparentGatewayApply) Apply(ctx context.Context, pending []string) error {
	apply.pending = append([]string(nil), pending...)
	return apply.err
}

type fakeGuideTransparentGatewayFacade struct {
	getResult *models.GuideGatewayRouterRequest
	getErr    error
	setResp   *models.SDKNormalResponse
	setErr    error
	setReqs   []models.GuideGatewayRouterRequest
	getCalls  int
}

func (facade *fakeGuideTransparentGatewayFacade) Get(ctx context.Context) (*models.GuideGatewayRouterRequest, error) {
	facade.getCalls++
	return facade.getResult, facade.getErr
}

func (facade *fakeGuideTransparentGatewayFacade) Set(ctx context.Context, req models.GuideGatewayRouterRequest) (*models.SDKNormalResponse, error) {
	facade.setReqs = append(facade.setReqs, req)
	return facade.setResp, facade.setErr
}

func TestGuideTransparentGatewayServiceGetBuildsLegacyResponseModel(t *testing.T) {
	service := GuideTransparentGatewayService{
		reader: &fakeGuideTransparentGatewayReader{
			snapshot: &GuideTransparentGatewaySnapshot{
				StaticLanIP: "192.168.50.1",
				SubnetMask:  "255.255.255.0",
				Gateway:     "192.168.50.254",
				StaticDNSIP: "223.5.5.5",
				EnableDhcp:  true,
			},
		},
	}

	result, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected transparent gateway get error: %v", err)
	}
	if result == nil || result.StaticLanIP != "192.168.50.1" || result.SubnetMask != "255.255.255.0" || result.Gateway != "192.168.50.254" || result.StaticDNSIP != "223.5.5.5" || !result.EnableDhcp {
		t.Fatalf("unexpected transparent gateway get result: %#v", result)
	}
}

func TestGuideTransparentGatewayServiceSetPreservesLegacyValidationAndPending(t *testing.T) {
	writer := &fakeGuideTransparentGatewayWriter{natChanged: true}
	apply := &fakeGuideTransparentGatewayApply{}
	service := GuideTransparentGatewayService{
		reader: &fakeGuideTransparentGatewayReader{},
		writer: writer,
		apply:  apply,
	}

	if _, err := service.Set(context.Background(), models.GuideGatewayRouterRequest{}); err == nil || err.Error() != "missing params" {
		t.Fatalf("expected missing params error, got %v", err)
	}

	resp, err := service.Set(context.Background(), models.GuideGatewayRouterRequest{
		StaticLanIP: "192.168.50.1",
		SubnetMask:  "255.255.255.0",
		Gateway:     "192.168.50.254",
		StaticDNSIP: "223.5.5.5",
		EnableDhcp:  true,
		Dhcp6c:      true,
		EnableNat:   true,
	})
	if err != nil {
		t.Fatalf("unexpected transparent gateway set error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected transparent gateway success response: %#v", resp)
	}
	if writer.dhcpEnabled == nil || !*writer.dhcpEnabled {
		t.Fatalf("expected DHCP enabled write, got %#v", writer.dhcpEnabled)
	}
	if writer.interfaceStaticIP != "192.168.50.1" || writer.interfaceMask != "255.255.255.0" || writer.interfaceGateway != "192.168.50.254" {
		t.Fatalf("unexpected interface input: ip=%q mask=%q gateway=%q", writer.interfaceStaticIP, writer.interfaceMask, writer.interfaceGateway)
	}
	if writer.dnsIP != "223.5.5.5" {
		t.Fatalf("unexpected DNS input: %q", writer.dnsIP)
	}
	if writer.lan6Enabled == nil || !*writer.lan6Enabled {
		t.Fatalf("expected lan6 enabled branch, got %#v", writer.lan6Enabled)
	}
	if writer.natEnabled == nil || !*writer.natEnabled {
		t.Fatalf("expected NAT write, got %#v", writer.natEnabled)
	}
	if !reflect.DeepEqual(apply.pending, []string{"firewall", "dhcp", "network"}) {
		t.Fatalf("unexpected pending configs: %v", apply.pending)
	}
}

func TestGuideTransparentGatewayServicePropagatesWriterAndApplyErrors(t *testing.T) {
	dhcpErr := errors.New("dhcp failed")
	service := GuideTransparentGatewayService{
		reader: &fakeGuideTransparentGatewayReader{},
		writer: &fakeGuideTransparentGatewayWriter{dhcpErr: dhcpErr},
		apply:  &fakeGuideTransparentGatewayApply{},
	}

	req := models.GuideGatewayRouterRequest{
		StaticLanIP: "192.168.50.1",
		SubnetMask:  "255.255.255.0",
		Gateway:     "192.168.50.254",
		StaticDNSIP: "223.5.5.5",
	}
	if _, err := service.Set(context.Background(), req); !errors.Is(err, dhcpErr) {
		t.Fatalf("expected DHCP writer error, got %v", err)
	}

	applyErr := errors.New("apply failed")
	service.writer = &fakeGuideTransparentGatewayWriter{}
	service.apply = &fakeGuideTransparentGatewayApply{err: applyErr}
	if _, err := service.Set(context.Background(), req); !errors.Is(err, applyErr) {
		t.Fatalf("expected apply error, got %v", err)
	}
}

func TestServiceBackendPostGuideGatewayRouterCompatibility(t *testing.T) {
	orig := newGuideTransparentGatewayServiceFacade
	defer func() { newGuideTransparentGatewayServiceFacade = orig }()

	success := models.ResponseSuccess(0)
	facade := &fakeGuideTransparentGatewayFacade{
		setResp: &models.SDKNormalResponse{Success: &success},
	}
	newGuideTransparentGatewayServiceFacade = func() guideTransparentGatewayFacade { return facade }

	postReq := httptest.NewRequest("POST", "/guide/gateway-router", strings.NewReader(`{"staticLanIp":"192.168.60.1","subnetMask":"255.255.255.0","gateway":"192.168.60.254","staticDnsIp":"1.1.1.1","enableDhcp":true,"dhcp6c":true,"enableNat":false}`))
	resp, err := (&ServiceBackend{}).PostGuideGatewayRouter(context.Background(), postReq)
	if err != nil || resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected PostGuideGatewayRouter response: resp=%#v err=%v", resp, err)
	}
	if len(facade.setReqs) != 1 || facade.setReqs[0].StaticLanIP != "192.168.60.1" || !facade.setReqs[0].EnableDhcp || !facade.setReqs[0].Dhcp6c || facade.setReqs[0].EnableNat {
		t.Fatalf("unexpected PostGuideGatewayRouter delegated reqs: %#v", facade.setReqs)
	}
}

func TestServiceBackendPostGuideGatewayRouterCompatibilityPropagateErrors(t *testing.T) {
	orig := newGuideTransparentGatewayServiceFacade
	defer func() { newGuideTransparentGatewayServiceFacade = orig }()

	serviceErr := errors.New("transparent gateway failed")
	newGuideTransparentGatewayServiceFacade = func() guideTransparentGatewayFacade {
		return &fakeGuideTransparentGatewayFacade{setErr: serviceErr}
	}

	postReq := httptest.NewRequest("POST", "/guide/gateway-router", strings.NewReader(`{"staticLanIp":"192.168.60.1","subnetMask":"255.255.255.0","gateway":"192.168.60.254","staticDnsIp":"1.1.1.1"}`))
	if _, err := (&ServiceBackend{}).PostGuideGatewayRouter(context.Background(), postReq); !errors.Is(err, serviceErr) {
		t.Fatalf("expected PostGuideGatewayRouter error, got %v", err)
	}
}

func TestTransparentGatewayCLICompatibility(t *testing.T) {
	orig := newGuideTransparentGatewayServiceFacade
	defer func() { newGuideTransparentGatewayServiceFacade = orig }()

	success := models.ResponseSuccess(0)
	facade := &fakeGuideTransparentGatewayFacade{
		getResult: &models.GuideGatewayRouterRequest{
			StaticLanIP: "192.168.70.1",
			SubnetMask:  "255.255.255.0",
			Gateway:     "192.168.70.254",
			StaticDNSIP: "8.8.8.8",
			EnableDhcp:  true,
		},
		setResp: &models.SDKNormalResponse{Success: &success},
	}
	newGuideTransparentGatewayServiceFacade = func() guideTransparentGatewayFacade { return facade }

	setReq := &models.GuideGatewayRouterRequest{
		StaticLanIP: "192.168.50.1",
		SubnetMask:  "255.255.255.0",
		Gateway:     "192.168.50.254",
		StaticDNSIP: "223.5.5.5",
		EnableDhcp:  true,
		Dhcp6c:      true,
		EnableNat:   true,
	}
	resp, err := SetTransparentGateway(context.Background(), setReq)
	if err != nil || resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected CLI SetTransparentGateway response: resp=%#v err=%v", resp, err)
	}
	if len(facade.setReqs) != 1 || !reflect.DeepEqual(facade.setReqs[0], *setReq) {
		t.Fatalf("unexpected CLI SetTransparentGateway delegated reqs: %#v", facade.setReqs)
	}

	getResp, err := GuideGetTransparentGateway()
	if err != nil || getResp == nil || getResp.StaticLanIP != "192.168.70.1" || getResp.SubnetMask != "255.255.255.0" || getResp.Gateway != "192.168.70.254" || getResp.StaticDNSIP != "8.8.8.8" || !getResp.EnableDhcp {
		t.Fatalf("unexpected CLI GuideGetTransparentGateway response: resp=%#v err=%v", getResp, err)
	}
	if facade.getCalls != 1 {
		t.Fatalf("expected one CLI get facade call, got %d", facade.getCalls)
	}
}
