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

type fakeGuideNetworkBasicsReader struct {
	defaultOutbound *GuideDefaultOutboundInterfaceSnapshot
	defaultErr      error
	dnsSnapshot     *GuideDNSConfigSnapshot
	dnsErr          error
	wanRuntime      *GuideWANRuntimeSnapshot
	wanRuntimeErr   error
	wanConfig       *GuideWANConfigSnapshot
	lanConfig       *GuideLANConfigSnapshot
}

func (reader *fakeGuideNetworkBasicsReader) ReadDefaultOutboundInterface(ctx context.Context) (*GuideDefaultOutboundInterfaceSnapshot, error) {
	if reader.defaultErr != nil {
		return nil, reader.defaultErr
	}
	return reader.defaultOutbound, nil
}

func (reader *fakeGuideNetworkBasicsReader) ReadDNSConfig(ctx context.Context) (*GuideDNSConfigSnapshot, error) {
	if reader.dnsErr != nil {
		return nil, reader.dnsErr
	}
	return reader.dnsSnapshot, nil
}

func (reader *fakeGuideNetworkBasicsReader) ReadWANConfig(ctx context.Context) *GuideWANConfigSnapshot {
	return reader.wanConfig
}

func (reader *fakeGuideNetworkBasicsReader) ReadWANRuntime(ctx context.Context, interfaceName string) (*GuideWANRuntimeSnapshot, error) {
	if reader.wanRuntimeErr != nil {
		return nil, reader.wanRuntimeErr
	}
	return reader.wanRuntime, nil
}

func (reader *fakeGuideNetworkBasicsReader) ReadLANConfig(ctx context.Context) *GuideLANConfigSnapshot {
	return reader.lanConfig
}

type fakeGuideNetworkBasicsWriter struct {
	dnsInput   GuideSetDNSConfigInput
	dnsErr     error
	wanInput   GuideSetWANInterfaceInput
	wanErr     error
	pppoeInput GuideSetPPPoEInput
	pppoeErr   error
	lanInput   GuideSetLANConfigInput
	lanPending []string
	lanErr     error
}

func (writer *fakeGuideNetworkBasicsWriter) SetDNSConfig(ctx context.Context, input GuideSetDNSConfigInput) error {
	writer.dnsInput = input
	return writer.dnsErr
}

func (writer *fakeGuideNetworkBasicsWriter) SetWANInterfaceMode(ctx context.Context, input GuideSetWANInterfaceInput) error {
	writer.wanInput = input
	return writer.wanErr
}

func (writer *fakeGuideNetworkBasicsWriter) SetPPPoE(ctx context.Context, input GuideSetPPPoEInput) error {
	writer.pppoeInput = input
	return writer.pppoeErr
}

func (writer *fakeGuideNetworkBasicsWriter) SetLANConfig(ctx context.Context, input GuideSetLANConfigInput) ([]string, error) {
	writer.lanInput = input
	if writer.lanErr != nil {
		return nil, writer.lanErr
	}
	return append([]string(nil), writer.lanPending...), nil
}

type fakeGuideNetworkBasicsApply struct {
	pending []string
	err     error
}

func (apply *fakeGuideNetworkBasicsApply) Apply(ctx context.Context, pending []string) error {
	apply.pending = append([]string(nil), pending...)
	return apply.err
}

type fakeGuideDNSConfigFacade struct {
	getResult *models.GuideDNSConfigResponseResult
	getErr    error
	setResult *models.GuideDNSConfigResponseResult
	setErr    error
	setInputs []models.GuideDNSConfigRequest
	getCalls  int
}

func (facade *fakeGuideDNSConfigFacade) Get(ctx context.Context) (*models.GuideDNSConfigResponseResult, error) {
	facade.getCalls++
	return facade.getResult, facade.getErr
}

func (facade *fakeGuideDNSConfigFacade) Set(ctx context.Context, req models.GuideDNSConfigRequest) (*models.GuideDNSConfigResponseResult, error) {
	facade.setInputs = append(facade.setInputs, req)
	return facade.setResult, facade.setErr
}

type fakeGuideDhcpClientFacade struct {
	getResult *models.GuideClientModeResponseResult
	getErr    error
	postResp  *models.SDKNormalResponse
	postErr   error
	postReqs  []models.GuideClientModeRequest
	getCalls  int
}

func (facade *fakeGuideDhcpClientFacade) Get(ctx context.Context) (*models.GuideClientModeResponseResult, *models.ResponseSuccess, models.ResponseError, error) {
	facade.getCalls++
	return facade.getResult, nil, "", facade.getErr
}

func (facade *fakeGuideDhcpClientFacade) Set(ctx context.Context, req models.GuideClientModeRequest) (*models.SDKNormalResponse, error) {
	facade.postReqs = append(facade.postReqs, req)
	return facade.postResp, facade.postErr
}

type fakeGuidePPPoEFacade struct {
	getResult *models.GuidePppoeStatusResponseResult
	getErr    error
	postResp  *models.SDKNormalResponse
	postErr   error
	postReqs  []models.GuidePppoeRequest
	getCalls  int
}

func (facade *fakeGuidePPPoEFacade) Get(ctx context.Context) (*models.GuidePppoeStatusResponseResult, *models.ResponseSuccess, models.ResponseError, error) {
	facade.getCalls++
	return facade.getResult, nil, "", facade.getErr
}

func (facade *fakeGuidePPPoEFacade) Set(ctx context.Context, req models.GuidePppoeRequest) (*models.SDKNormalResponse, error) {
	facade.postReqs = append(facade.postReqs, req)
	return facade.postResp, facade.postErr
}

type fakeGuideLanSettingFacade struct {
	getResult *models.GuideLanSettingResponseResult
	getErr    error
	postResp  *models.SDKNormalResponse
	postErr   error
	postReqs  []models.GuideLanSettingRequest
	getCalls  int
}

func (facade *fakeGuideLanSettingFacade) Get(ctx context.Context) (*models.GuideLanSettingResponseResult, error) {
	facade.getCalls++
	return facade.getResult, facade.getErr
}

func (facade *fakeGuideLanSettingFacade) Set(ctx context.Context, req models.GuideLanSettingRequest) (*models.SDKNormalResponse, error) {
	facade.postReqs = append(facade.postReqs, req)
	return facade.postResp, facade.postErr
}

func TestGuideDNSConfigServiceGetReturnsSnapshot(t *testing.T) {
	t.Parallel()

	service := GuideDNSConfigService{
		reader: &fakeGuideNetworkBasicsReader{
			dnsSnapshot: &GuideDNSConfigSnapshot{
				InterfaceName: "wan",
				DNSProto:      "manual",
				ManualDNSIP:   []string{"1.1.1.1"},
			},
		},
	}

	result, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected DNS get error: %v", err)
	}
	if result == nil || result.InterfaceName != "wan" || result.DNSProto != "manual" || len(result.ManualDNSIP) != 1 || result.ManualDNSIP[0] != "1.1.1.1" {
		t.Fatalf("unexpected DNS get result: %#v", result)
	}
}

func TestGuideDNSConfigServiceSetValidatesAndPreservesStaticAutoRejection(t *testing.T) {
	t.Parallel()

	service := GuideDNSConfigService{
		reader: &fakeGuideNetworkBasicsReader{
			defaultOutbound: &GuideDefaultOutboundInterfaceSnapshot{
				InterfaceName: "wan",
			},
			wanRuntime: &GuideWANRuntimeSnapshot{},
			defaultErr: nil,
		},
		writer: &fakeGuideNetworkBasicsWriter{},
		apply:  &fakeGuideNetworkBasicsApply{},
	}

	if _, err := service.Set(context.Background(), models.GuideDNSConfigRequest{}); err == nil || err.Error() != "missing params" {
		t.Fatalf("expected missing params error, got %v", err)
	}

	service.reader = &fakeGuideNetworkBasicsReader{
		defaultOutbound: &GuideDefaultOutboundInterfaceSnapshot{
			InterfaceName: "wan",
			Proto:         "static",
		},
		wanRuntime: &GuideWANRuntimeSnapshot{},
	}
	if _, err := service.Set(context.Background(), models.GuideDNSConfigRequest{
		DNSProto: "auto",
	}); err == nil || err.Error() != "dns must be set when using static proto" {
		t.Fatalf("expected static-auto rejection, got %v", err)
	}
}

func TestGuideDNSConfigServiceSetWritesAndApplies(t *testing.T) {
	t.Parallel()

	writer := &fakeGuideNetworkBasicsWriter{}
	apply := &fakeGuideNetworkBasicsApply{}
	service := GuideDNSConfigService{
		reader: &fakeGuideNetworkBasicsReader{
			defaultOutbound: &GuideDefaultOutboundInterfaceSnapshot{
				InterfaceName: "wan",
				Proto:         "dhcp",
			},
		},
		writer: writer,
		apply:  apply,
	}

	result, err := service.Set(context.Background(), models.GuideDNSConfigRequest{
		DNSProto:    "manual",
		ManualDNSIP: []string{"1.1.1.1", "8.8.8.8"},
	})
	if err != nil {
		t.Fatalf("unexpected DNS set error: %v", err)
	}
	if writer.dnsInput.InterfaceName != "wan" || writer.dnsInput.DNSProto != "manual" || len(writer.dnsInput.ManualDNSIP) != 2 {
		t.Fatalf("unexpected DNS write input: %#v", writer.dnsInput)
	}
	if len(apply.pending) != 1 || apply.pending[0] != "network" {
		t.Fatalf("unexpected apply pending list: %v", apply.pending)
	}
	if result == nil || result.InterfaceName != "wan" || result.DNSProto != "manual" || len(result.ManualDNSIP) != 2 {
		t.Fatalf("unexpected DNS set result: %#v", result)
	}
}

func TestGuideDNSConfigServiceSetPropagatesWriterAndApplyErrors(t *testing.T) {
	t.Parallel()

	writerErr := errors.New("write DNS failed")
	service := GuideDNSConfigService{
		reader: &fakeGuideNetworkBasicsReader{
			defaultOutbound: &GuideDefaultOutboundInterfaceSnapshot{
				InterfaceName: "wan",
				Proto:         "dhcp",
			},
		},
		writer: &fakeGuideNetworkBasicsWriter{dnsErr: writerErr},
		apply:  &fakeGuideNetworkBasicsApply{},
	}
	if _, err := service.Set(context.Background(), models.GuideDNSConfigRequest{
		DNSProto: "auto",
	}); !errors.Is(err, writerErr) {
		t.Fatalf("expected writer error, got %v", err)
	}

	applyErr := errors.New("apply DNS failed")
	service.writer = &fakeGuideNetworkBasicsWriter{}
	service.apply = &fakeGuideNetworkBasicsApply{err: applyErr}
	if _, err := service.Set(context.Background(), models.GuideDNSConfigRequest{
		DNSProto: "auto",
	}); !errors.Is(err, applyErr) {
		t.Fatalf("expected apply error, got %v", err)
	}
}

func TestServiceBackendGetGuideDnsConfigCompatibilityDelegatesToService(t *testing.T) {
	originalFactory := newGuideDNSConfigServiceFacade
	defer func() {
		newGuideDNSConfigServiceFacade = originalFactory
	}()

	facade := &fakeGuideDNSConfigFacade{
		getResult: &models.GuideDNSConfigResponseResult{
			InterfaceName: "wan",
			DNSProto:      "manual",
			ManualDNSIP:   []string{"1.1.1.1"},
		},
	}
	newGuideDNSConfigServiceFacade = func() guideDNSConfigFacade {
		return facade
	}

	resp, err := (&ServiceBackend{}).GetGuideDnsConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected get wrapper error: %v", err)
	}
	if facade.getCalls != 1 {
		t.Fatalf("expected one get call, got %d", facade.getCalls)
	}
	if resp == nil || resp.Result == nil || resp.Result.InterfaceName != "wan" || resp.Result.DNSProto != "manual" {
		t.Fatalf("unexpected get wrapper response: %#v", resp)
	}
}

func TestServiceBackendPostGuideDnsConfigCompatibilityDelegatesToService(t *testing.T) {
	originalFactory := newGuideDNSConfigServiceFacade
	defer func() {
		newGuideDNSConfigServiceFacade = originalFactory
	}()

	facade := &fakeGuideDNSConfigFacade{
		setResult: &models.GuideDNSConfigResponseResult{
			InterfaceName: "wan",
			DNSProto:      "manual",
			ManualDNSIP:   []string{"1.1.1.1"},
		},
	}
	newGuideDNSConfigServiceFacade = func() guideDNSConfigFacade {
		return facade
	}

	req := httptest.NewRequest("POST", "/guide/dns", strings.NewReader(`{"dnsProto":"manual","manualDnsIp":["1.1.1.1"]}`))
	resp, err := (&ServiceBackend{}).PostGuideDnsConfig(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected post wrapper error: %v", err)
	}
	if len(facade.setInputs) != 1 || facade.setInputs[0].DNSProto != "manual" || len(facade.setInputs[0].ManualDNSIP) != 1 || facade.setInputs[0].ManualDNSIP[0] != "1.1.1.1" {
		t.Fatalf("unexpected set wrapper inputs: %#v", facade.setInputs)
	}
	if resp == nil || resp.Result == nil || resp.Result.DNSProto != "manual" {
		t.Fatalf("unexpected post wrapper response: %#v", resp)
	}
}

func TestServiceBackendGuideDnsConfigCompatibilityPropagatesServiceErrors(t *testing.T) {
	originalFactory := newGuideDNSConfigServiceFacade
	defer func() {
		newGuideDNSConfigServiceFacade = originalFactory
	}()

	serviceErr := errors.New("dns service failed")
	newGuideDNSConfigServiceFacade = func() guideDNSConfigFacade {
		return &fakeGuideDNSConfigFacade{getErr: serviceErr, setErr: serviceErr}
	}

	if _, err := (&ServiceBackend{}).GetGuideDnsConfig(context.Background()); !errors.Is(err, serviceErr) {
		t.Fatalf("expected get wrapper error, got %v", err)
	}
	req := httptest.NewRequest("POST", "/guide/dns", strings.NewReader(`{"dnsProto":"auto"}`))
	if _, err := (&ServiceBackend{}).PostGuideDnsConfig(context.Background(), req); !errors.Is(err, serviceErr) {
		t.Fatalf("expected post wrapper error, got %v", err)
	}
}

func TestGuideDhcpClientServiceGetHandlesWanPresenceAndRuntimeFallback(t *testing.T) {
	service := GuideDhcpClientService{
		reader: &fakeGuideNetworkBasicsReader{
			wanConfig: &GuideWANConfigSnapshot{
				Exists:      true,
				WanProto:    "dhcp",
				DNSProto:    "manual",
				ManualDNSIP: []string{"1.1.1.1"},
			},
			wanRuntime: &GuideWANRuntimeSnapshot{
				StaticIP:   "10.0.0.2",
				SubnetMask: "255.255.255.0",
				Gateway:    "10.0.0.1",
			},
		},
	}

	result, success, respErr, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if success != nil || respErr != "" {
		t.Fatalf("expected normal result, got success=%v err=%v", success, respErr)
	}
	if result == nil || result.WanProto != "dhcp" || result.StaticIP != "10.0.0.2" || result.SubnetMask != "255.255.255.0" || result.Gateway != "10.0.0.1" || result.DNSProto != "manual" || len(result.ManualDNSIP) != 1 {
		t.Fatalf("unexpected DHCP client get result: %#v", result)
	}

	service.reader = &fakeGuideNetworkBasicsReader{
		wanConfig: &GuideWANConfigSnapshot{Exists: false},
	}
	result, success, respErr, err = service.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected missing-wan get error: %v", err)
	}
	if result != nil || success == nil || *success != NetworkErrorWanNotExists || string(respErr) != string(NetworkErrorMessageWanNotExists) {
		t.Fatalf("unexpected missing-wan response: result=%#v success=%v err=%v", result, success, respErr)
	}
}

func TestGuideDhcpClientServiceSetPreservesLegacyValidationAndPendingApply(t *testing.T) {
	originalDeleteLan6 := writeGuideNetworkBasicsDeleteLan6
	originalSetMasq := writeGuideNetworkBasicsSetLanMasq
	originalEnableDhcp := writeGuideNetworkBasicsEnableLanDHCP
	defer func() {
		writeGuideNetworkBasicsDeleteLan6 = originalDeleteLan6
		writeGuideNetworkBasicsSetLanMasq = originalSetMasq
		writeGuideNetworkBasicsEnableLanDHCP = originalEnableDhcp
	}()

	var enableCalls int
	writeGuideNetworkBasicsDeleteLan6 = func(ctx context.Context) error { return nil }
	writeGuideNetworkBasicsSetLanMasq = func(ctx context.Context, enable bool) bool { return false }
	writeGuideNetworkBasicsEnableLanDHCP = func(ctx context.Context) { enableCalls++ }

	service := GuideDhcpClientService{
		reader: &fakeGuideNetworkBasicsReader{
			wanConfig: &GuideWANConfigSnapshot{Exists: true},
		},
		writer: &fakeGuideNetworkBasicsWriter{lanPending: []string{"dhcp", "network"}},
		apply:  &fakeGuideNetworkBasicsApply{},
	}

	if _, err := service.Set(context.Background(), models.GuideClientModeRequest{WanProto: "bad"}); err == nil || err.Error() != "WanProto should be static or dhcp" {
		t.Fatalf("expected invalid proto error, got %v", err)
	}

	resp, err := service.Set(context.Background(), models.GuideClientModeRequest{
		WanProto:   "static",
		DNSProto:   "auto",
		SubnetMask: "255.255.255.0",
		Gateway:    "10.0.0.1",
	})
	if err != nil {
		t.Fatalf("unexpected static-auto response error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != NetworkErrorDnsNotSetting || string(resp.Error) != "静态IP地址，dns必须手动配置" {
		t.Fatalf("unexpected static-auto response: %#v", resp)
	}

	writer := &fakeGuideNetworkBasicsWriter{lanPending: []string{"dhcp", "network"}}
	apply := &fakeGuideNetworkBasicsApply{}
	service.writer = writer
	service.apply = apply
	resp, err = service.Set(context.Background(), models.GuideClientModeRequest{
		WanProto:      "dhcp",
		DNSProto:      "manual",
		ManualDNSIP:   []string{"1.1.1.1"},
		EnableLanDhcp: true,
	})
	if err != nil {
		t.Fatalf("unexpected DHCP client set error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected DHCP client success response: %#v", resp)
	}
	if writer.wanInput.InterfaceName != "wan" || writer.wanInput.WanProto != "dhcp" {
		t.Fatalf("unexpected WAN writer input: %#v", writer.wanInput)
	}
	if writer.dnsInput.InterfaceName != "wan" || writer.dnsInput.DNSProto != "manual" || len(writer.dnsInput.ManualDNSIP) != 1 {
		t.Fatalf("unexpected DNS writer input: %#v", writer.dnsInput)
	}
	if enableCalls != 1 {
		t.Fatalf("expected enable DHCP helper once, got %d", enableCalls)
	}
	if !reflect.DeepEqual(apply.pending, []string{"dhcp", "network"}) {
		t.Fatalf("unexpected DHCP client pending apply list: %v", apply.pending)
	}
}

func TestGuideDhcpClientServiceSetPropagatesErrors(t *testing.T) {
	originalDeleteLan6 := writeGuideNetworkBasicsDeleteLan6
	originalSetMasq := writeGuideNetworkBasicsSetLanMasq
	defer func() {
		writeGuideNetworkBasicsDeleteLan6 = originalDeleteLan6
		writeGuideNetworkBasicsSetLanMasq = originalSetMasq
	}()

	writeGuideNetworkBasicsDeleteLan6 = func(ctx context.Context) error { return nil }
	writeGuideNetworkBasicsSetLanMasq = func(ctx context.Context, enable bool) bool { return false }

	wanErr := errors.New("write wan failed")
	service := GuideDhcpClientService{
		reader: &fakeGuideNetworkBasicsReader{wanConfig: &GuideWANConfigSnapshot{Exists: true}},
		writer: &fakeGuideNetworkBasicsWriter{wanErr: wanErr},
		apply:  &fakeGuideNetworkBasicsApply{},
	}
	if _, err := service.Set(context.Background(), models.GuideClientModeRequest{
		WanProto: "dhcp",
		DNSProto: "auto",
	}); !errors.Is(err, wanErr) {
		t.Fatalf("expected WAN write error, got %v", err)
	}

	applyErr := errors.New("apply wan failed")
	service.writer = &fakeGuideNetworkBasicsWriter{lanPending: []string{"network"}}
	service.apply = &fakeGuideNetworkBasicsApply{err: applyErr}
	if _, err := service.Set(context.Background(), models.GuideClientModeRequest{
		WanProto: "dhcp",
		DNSProto: "auto",
	}); !errors.Is(err, applyErr) {
		t.Fatalf("expected apply error, got %v", err)
	}
}

func TestGuidePPPoEServiceGetAndSetPreserveLegacySemantics(t *testing.T) {
	originalDeleteLan6 := writeGuideNetworkBasicsDeleteLan6
	originalSetMasq := writeGuideNetworkBasicsSetLanMasq
	originalEnableDhcp := writeGuideNetworkBasicsEnableLanDHCP
	defer func() {
		writeGuideNetworkBasicsDeleteLan6 = originalDeleteLan6
		writeGuideNetworkBasicsSetLanMasq = originalSetMasq
		writeGuideNetworkBasicsEnableLanDHCP = originalEnableDhcp
	}()

	var enableCalls int
	writeGuideNetworkBasicsDeleteLan6 = func(ctx context.Context) error { return nil }
	writeGuideNetworkBasicsSetLanMasq = func(ctx context.Context, enable bool) bool { return true }
	writeGuideNetworkBasicsEnableLanDHCP = func(ctx context.Context) { enableCalls++ }

	service := GuidePPPoEService{
		reader: &fakeGuideNetworkBasicsReader{
			wanConfig: &GuideWANConfigSnapshot{
				Exists:        true,
				PPPoEAccount:  "pppoe-user",
				PPPoEPassword: "pppoe-pass",
			},
		},
		writer: &fakeGuideNetworkBasicsWriter{lanPending: []string{"network"}},
		apply:  &fakeGuideNetworkBasicsApply{},
	}

	getResult, success, respErr, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected PPPOE get error: %v", err)
	}
	if success != nil || respErr != "" {
		t.Fatalf("expected normal PPPOE result, got success=%v err=%v", success, respErr)
	}
	if getResult == nil || getResult.Account != "pppoe-user" || getResult.Password != "pppoe-pass" {
		t.Fatalf("unexpected PPPOE get result: %#v", getResult)
	}

	service.reader = &fakeGuideNetworkBasicsReader{wanConfig: &GuideWANConfigSnapshot{Exists: false}}
	getResult, success, respErr, err = service.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected missing-wan PPPOE error: %v", err)
	}
	if getResult != nil || success == nil || *success != NetworkErrorWanNotExists || string(respErr) != string(NetworkErrorMessageWanNotExists) {
		t.Fatalf("unexpected missing-wan PPPOE response: result=%#v success=%v err=%v", getResult, success, respErr)
	}

	writer := &fakeGuideNetworkBasicsWriter{lanPending: []string{"firewall", "dhcp", "network"}}
	apply := &fakeGuideNetworkBasicsApply{}
	service.reader = &fakeGuideNetworkBasicsReader{wanConfig: &GuideWANConfigSnapshot{Exists: true}}
	service.writer = writer
	service.apply = apply
	resp, err := service.Set(context.Background(), models.GuidePppoeRequest{
		Account:       "pppoe-user",
		Password:      "pppoe-pass",
		EnableLanDhcp: true,
	})
	if err != nil {
		t.Fatalf("unexpected PPPOE set error: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected PPPOE response")
	}
	if writer.pppoeInput.Account != "pppoe-user" || writer.pppoeInput.Password != "pppoe-pass" {
		t.Fatalf("unexpected PPPOE writer input: %#v", writer.pppoeInput)
	}
	if enableCalls != 1 {
		t.Fatalf("expected enable DHCP helper once, got %d", enableCalls)
	}
	if !reflect.DeepEqual(apply.pending, []string{"firewall", "dhcp", "network"}) {
		t.Fatalf("unexpected PPPOE pending apply list: %v", apply.pending)
	}
}

func TestServiceBackendGuideDhcpClientAndPPPoECompatibilityWrappers(t *testing.T) {
	origDhcp := newGuideDhcpClientServiceFacade
	origPPPoE := newGuidePPPoEServiceFacade
	defer func() {
		newGuideDhcpClientServiceFacade = origDhcp
		newGuidePPPoEServiceFacade = origPPPoE
	}()

	dhcpFacade := &fakeGuideDhcpClientFacade{
		getResult: &models.GuideClientModeResponseResult{WanProto: "dhcp"},
		postResp:  &models.SDKNormalResponse{Success: func() *models.ResponseSuccess { v := models.ResponseSuccess(0); return &v }()},
	}
	pppoeFacade := &fakeGuidePPPoEFacade{
		getResult: &models.GuidePppoeStatusResponseResult{Account: "user"},
		postResp:  &models.SDKNormalResponse{},
	}
	newGuideDhcpClientServiceFacade = func() guideDhcpClientFacade { return dhcpFacade }
	newGuidePPPoEServiceFacade = func() guidePPPoEFacade { return pppoeFacade }
	backend := &ServiceBackend{}

	dhcpGetResp, err := backend.GetGuideClientMode(context.Background())
	if err != nil || dhcpGetResp == nil || dhcpGetResp.Result == nil || dhcpGetResp.Result.WanProto != "dhcp" {
		t.Fatalf("unexpected DHCP get wrapper response: resp=%#v err=%v", dhcpGetResp, err)
	}
	dhcpReq := httptest.NewRequest("POST", "/guide/dhcp-client", strings.NewReader(`{"wanProto":"dhcp","dnsProto":"auto"}`))
	if _, err := backend.PostGuideClientMode(context.Background(), dhcpReq); err != nil {
		t.Fatalf("unexpected DHCP post wrapper error: %v", err)
	}
	if len(dhcpFacade.postReqs) != 1 || dhcpFacade.postReqs[0].WanProto != "dhcp" {
		t.Fatalf("unexpected DHCP wrapper requests: %#v", dhcpFacade.postReqs)
	}

	pppoeGetResp, err := backend.GetGuidePppoe(context.Background())
	if err != nil || pppoeGetResp == nil || pppoeGetResp.Result == nil || pppoeGetResp.Result.Account != "user" {
		t.Fatalf("unexpected PPPOE get wrapper response: resp=%#v err=%v", pppoeGetResp, err)
	}
	pppoeReq := httptest.NewRequest("POST", "/guide/pppoe", strings.NewReader(`{"account":"user","password":"pw","enableLanDhcp":true}`))
	if _, err := backend.PostGuidePppoe(context.Background(), pppoeReq); err != nil {
		t.Fatalf("unexpected PPPOE post wrapper error: %v", err)
	}
	if len(pppoeFacade.postReqs) != 1 || pppoeFacade.postReqs[0].Account != "user" || !pppoeFacade.postReqs[0].EnableLanDhcp {
		t.Fatalf("unexpected PPPOE wrapper requests: %#v", pppoeFacade.postReqs)
	}
}

func TestServiceBackendGuideDhcpClientAndPPPoECompatibilityPropagateServiceErrors(t *testing.T) {
	origDhcp := newGuideDhcpClientServiceFacade
	origPPPoE := newGuidePPPoEServiceFacade
	defer func() {
		newGuideDhcpClientServiceFacade = origDhcp
		newGuidePPPoEServiceFacade = origPPPoE
	}()

	serviceErr := errors.New("wan mode service failed")
	newGuideDhcpClientServiceFacade = func() guideDhcpClientFacade {
		return &fakeGuideDhcpClientFacade{getErr: serviceErr, postErr: serviceErr}
	}
	newGuidePPPoEServiceFacade = func() guidePPPoEFacade {
		return &fakeGuidePPPoEFacade{getErr: serviceErr, postErr: serviceErr}
	}
	backend := &ServiceBackend{}

	if _, err := backend.GetGuideClientMode(context.Background()); !errors.Is(err, serviceErr) {
		t.Fatalf("expected DHCP get wrapper error, got %v", err)
	}
	dhcpReq := httptest.NewRequest("POST", "/guide/dhcp-client", strings.NewReader(`{"wanProto":"dhcp","dnsProto":"auto"}`))
	if _, err := backend.PostGuideClientMode(context.Background(), dhcpReq); !errors.Is(err, serviceErr) {
		t.Fatalf("expected DHCP post wrapper error, got %v", err)
	}
	if _, err := backend.GetGuidePppoe(context.Background()); !errors.Is(err, serviceErr) {
		t.Fatalf("expected PPPOE get wrapper error, got %v", err)
	}
	pppoeReq := httptest.NewRequest("POST", "/guide/pppoe", strings.NewReader(`{"account":"user","password":"pw"}`))
	if _, err := backend.PostGuidePppoe(context.Background(), pppoeReq); !errors.Is(err, serviceErr) {
		t.Fatalf("expected PPPOE post wrapper error, got %v", err)
	}
}

func TestGuideLanSettingServiceGetAndSetPreserveLegacySemantics(t *testing.T) {
	writer := &fakeGuideNetworkBasicsWriter{}
	apply := &fakeGuideNetworkBasicsApply{}
	service := GuideLanSettingService{
		reader: &fakeGuideNetworkBasicsReader{
			lanConfig: &GuideLANConfigSnapshot{
				LanIP:      "192.168.100.1",
				NetMask:    "255.255.255.0",
				EnableDhcp: true,
				DhcpStart:  "192.168.100.100",
				DhcpEnd:    "192.168.100.248",
			},
		},
		writer: writer,
		apply:  apply,
	}

	getResult, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected LAN setting get error: %v", err)
	}
	if getResult == nil || getResult.LanIP != "192.168.100.1" || getResult.NetMask != "255.255.255.0" || !getResult.EnableDhcp || getResult.DhcpStart != "192.168.100.100" || getResult.DhcpEnd != "192.168.100.248" {
		t.Fatalf("unexpected LAN setting get result: %#v", getResult)
	}

	if _, err := service.Set(context.Background(), models.GuideLanSettingRequest{}); err == nil || err.Error() != "missing params" {
		t.Fatalf("expected missing params error, got %v", err)
	}
	if _, err := service.Set(context.Background(), models.GuideLanSettingRequest{
		LanIP:      "192.168.100.1",
		NetMask:    "255.255.255.0",
		EnableDhcp: true,
		DhcpStart:  "bad-ip",
		DhcpEnd:    "192.168.100.200",
	}); err == nil || err.Error() != "IP池起始或结束地址错误" {
		t.Fatalf("expected invalid DHCP pool error, got %v", err)
	}

	writer.lanPending = []string{"dhcp", "network"}
	resp, err := service.Set(context.Background(), models.GuideLanSettingRequest{
		LanIP:      "192.168.100.1",
		NetMask:    "255.255.255.0",
		EnableDhcp: true,
		DhcpStart:  "192.168.100.100",
		DhcpEnd:    "192.168.100.200",
	})
	if err != nil {
		t.Fatalf("unexpected LAN setting set error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected LAN setting success response: %#v", resp)
	}
	if writer.lanInput.LanIP != "192.168.100.1" || writer.lanInput.NetMask != "255.255.255.0" || !writer.lanInput.EnableDhcp || writer.lanInput.DhcpStart != "192.168.100.100" || writer.lanInput.DhcpEnd != "192.168.100.200" {
		t.Fatalf("unexpected LAN writer input: %#v", writer.lanInput)
	}
	if !reflect.DeepEqual(apply.pending, []string{"dhcp"}) {
		t.Fatalf("unexpected LAN setting pending apply list: %v", apply.pending)
	}

	apply.pending = nil
	writer.lanPending = []string{"network"}
	resp, err = service.Set(context.Background(), models.GuideLanSettingRequest{
		LanIP:   "192.168.50.1",
		NetMask: "255.255.255.0",
	})
	if err != nil {
		t.Fatalf("unexpected LAN setting no-dhcp set error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected LAN setting no-dhcp success response: %#v", resp)
	}
	if len(apply.pending) != 0 {
		t.Fatalf("expected no shared apply for network-only LAN update, got %v", apply.pending)
	}
}

func TestGuideLanSettingServicePropagatesExpectedErrors(t *testing.T) {
	writerErr := errors.New("set lan failed")
	service := GuideLanSettingService{
		reader: &fakeGuideNetworkBasicsReader{
			lanConfig: &GuideLANConfigSnapshot{},
		},
		writer: &fakeGuideNetworkBasicsWriter{lanErr: writerErr},
		apply:  &fakeGuideNetworkBasicsApply{},
	}

	if _, err := service.Set(context.Background(), models.GuideLanSettingRequest{
		LanIP:      "192.168.100.1",
		NetMask:    "255.255.255.0",
		EnableDhcp: true,
		DhcpStart:  "192.168.100.100",
		DhcpEnd:    "192.168.100.200",
	}); !errors.Is(err, writerErr) {
		t.Fatalf("expected writer error, got %v", err)
	}

	applyErr := errors.New("apply lan failed")
	service.writer = &fakeGuideNetworkBasicsWriter{lanPending: []string{"dhcp", "network"}}
	service.apply = &fakeGuideNetworkBasicsApply{err: applyErr}
	if _, err := service.Set(context.Background(), models.GuideLanSettingRequest{
		LanIP:      "192.168.100.1",
		NetMask:    "255.255.255.0",
		EnableDhcp: true,
		DhcpStart:  "192.168.100.100",
		DhcpEnd:    "192.168.100.200",
	}); !errors.Is(err, applyErr) {
		t.Fatalf("expected apply error, got %v", err)
	}
}

func TestServiceBackendGuideLanSettingCompatibilityWrappers(t *testing.T) {
	orig := newGuideLanSettingServiceFacade
	defer func() { newGuideLanSettingServiceFacade = orig }()

	facade := &fakeGuideLanSettingFacade{
		getResult: &models.GuideLanSettingResponseResult{LanIP: "192.168.50.1"},
		postResp:  &models.SDKNormalResponse{Success: func() *models.ResponseSuccess { v := models.ResponseSuccess(0); return &v }()},
	}
	newGuideLanSettingServiceFacade = func() guideLanSettingFacade { return facade }
	backend := &ServiceBackend{}

	getResp, err := backend.GetGuideLan(context.Background())
	if err != nil || getResp == nil || getResp.Result == nil || getResp.Result.LanIP != "192.168.50.1" {
		t.Fatalf("unexpected LAN get wrapper response: resp=%#v err=%v", getResp, err)
	}
	postReq := httptest.NewRequest("POST", "/guide/lan-setting", strings.NewReader(`{"lanIp":"192.168.50.1","netMask":"255.255.255.0","enableDhcp":true,"dhcpStart":"192.168.50.100","dhcpEnd":"192.168.50.200"}`))
	if _, err := backend.PostGuideLan(context.Background(), postReq); err != nil {
		t.Fatalf("unexpected LAN post wrapper error: %v", err)
	}
	if len(facade.postReqs) != 1 || facade.postReqs[0].LanIP != "192.168.50.1" || !facade.postReqs[0].EnableDhcp {
		t.Fatalf("unexpected LAN wrapper requests: %#v", facade.postReqs)
	}
}

func TestServiceBackendGuideLanSettingCompatibilityPropagatesServiceErrors(t *testing.T) {
	orig := newGuideLanSettingServiceFacade
	defer func() { newGuideLanSettingServiceFacade = orig }()

	serviceErr := errors.New("lan setting service failed")
	newGuideLanSettingServiceFacade = func() guideLanSettingFacade {
		return &fakeGuideLanSettingFacade{getErr: serviceErr, postErr: serviceErr}
	}
	backend := &ServiceBackend{}

	if _, err := backend.GetGuideLan(context.Background()); !errors.Is(err, serviceErr) {
		t.Fatalf("expected LAN get wrapper error, got %v", err)
	}
	postReq := httptest.NewRequest("POST", "/guide/lan-setting", strings.NewReader(`{"lanIp":"192.168.50.1","netMask":"255.255.255.0"}`))
	if _, err := backend.PostGuideLan(context.Background(), postReq); !errors.Is(err, serviceErr) {
		t.Fatalf("expected LAN post wrapper error, got %v", err)
	}
}

func TestGuideGetLanSettingCLICompatibility(t *testing.T) {
	orig := newGuideLanSettingServiceFacade
	defer func() { newGuideLanSettingServiceFacade = orig }()

	facade := &fakeGuideLanSettingFacade{
		getResult: &models.GuideLanSettingResponseResult{LanIP: "192.168.50.1"},
	}
	newGuideLanSettingServiceFacade = func() guideLanSettingFacade { return facade }

	resp, err := GuideGetLanSetting(context.Background())
	if err != nil || resp == nil || resp.Result == nil || resp.Result.LanIP != "192.168.50.1" {
		t.Fatalf("unexpected CLI LAN get response: resp=%#v err=%v", resp, err)
	}
	if facade.getCalls != 1 {
		t.Fatalf("expected one CLI get call, got %d", facade.getCalls)
	}
}
