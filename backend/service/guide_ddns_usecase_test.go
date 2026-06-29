package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func reqWithBody(t *testing.T, body string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, "/guide/test", strings.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected request build error: %v", err)
	}
	return req
}

type fakeGuideDDNSWriter struct {
	ddnstoEnableInput  *GuideDdnstoEnableInput
	ddnstoEnableStderr string
	ddnstoEnableErr    error

	ddnstoAddressInput  *GuideDdnstoAddressInput
	ddnstoAddressStderr string
	ddnstoAddressErr    error

	ddnsCmds []string
	ddnsErr  error

	startConfigName string
	startErr        error
}

type fakeGuideDDNSReader struct {
	pending     bool
	pendingErr  error
	outbound    *GuideDDNSOutboundSnapshot
	outboundErr error
	publicIPv4  bool
	publicIPv6  bool
}

func (writer *fakeGuideDDNSWriter) EnableDdnsto(ctx context.Context, input GuideDdnstoEnableInput) (string, error) {
	copied := input
	writer.ddnstoEnableInput = &copied
	return writer.ddnstoEnableStderr, writer.ddnstoEnableErr
}

func (writer *fakeGuideDDNSWriter) UpdateDdnstoAddress(ctx context.Context, input GuideDdnstoAddressInput) (string, error) {
	copied := input
	writer.ddnstoAddressInput = &copied
	return writer.ddnstoAddressStderr, writer.ddnstoAddressErr
}

func (writer *fakeGuideDDNSWriter) ApplyDDNSConfig(ctx context.Context, cmds []string) error {
	writer.ddnsCmds = append([]string(nil), cmds...)
	return writer.ddnsErr
}

func (writer *fakeGuideDDNSWriter) StartDDNSService(ctx context.Context, configName string) error {
	writer.startConfigName = configName
	return writer.startErr
}

func (reader *fakeGuideDDNSReader) ReadDDNSPendingChanges(ctx context.Context, sessionID string) (bool, error) {
	return reader.pending, reader.pendingErr
}

func (reader *fakeGuideDDNSReader) ReadOutboundInterfaces(ctx context.Context) (*GuideDDNSOutboundSnapshot, error) {
	return reader.outbound, reader.outboundErr
}

func (reader *fakeGuideDDNSReader) IsPublicIPv4(ip string) bool {
	return reader.publicIPv4
}

func (reader *fakeGuideDDNSReader) IsPublicIPv6(ip string) bool {
	return reader.publicIPv6
}

func (reader *fakeGuideDDNSReader) ReadDdnstoConfig(ctx context.Context) (*GuideDdnstoConfigSnapshot, error) {
	return nil, nil
}

func TestGuideDdnstoEnableServiceBuildsSuccessResponse(t *testing.T) {
	t.Parallel()

	writer := &fakeGuideDDNSWriter{}
	service := GuideDdnstoEnableService{writer: writer}

	resp, err := service.Enable(context.Background(), GuideDdnstoEnableInput{Token: "token-abc"})
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if writer.ddnstoEnableInput == nil || writer.ddnstoEnableInput.Token != "token-abc" {
		t.Fatalf("unexpected writer input: %#v", writer.ddnstoEnableInput)
	}
	if resp == nil || resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("unexpected success response: %#v", resp)
	}
}

func TestGuideDdnstoEnableServiceMapsLegacyErrorWording(t *testing.T) {
	t.Parallel()

	writer := &fakeGuideDDNSWriter{
		ddnstoEnableStderr: " boom",
		ddnstoEnableErr:    errors.New("restart failed"),
	}
	service := GuideDdnstoEnableService{writer: writer}

	if _, err := service.Enable(context.Background(), GuideDdnstoEnableInput{Token: "token-abc"}); err == nil || err.Error() != "ddnsto启动失败 boom" {
		t.Fatalf("unexpected enable error: %v", err)
	}
}

func TestGuideDdnstoAddressServiceBuildsSuccessResponse(t *testing.T) {
	t.Parallel()

	writer := &fakeGuideDDNSWriter{}
	service := GuideDdnstoAddressService{writer: writer}

	resp, err := service.UpdateAddress(context.Background(), GuideDdnstoAddressInput{Address: "https://demo.example.com"})
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if writer.ddnstoAddressInput == nil || writer.ddnstoAddressInput.Address != "https://demo.example.com" {
		t.Fatalf("unexpected address writer input: %#v", writer.ddnstoAddressInput)
	}
	if resp == nil || resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("unexpected success response: %#v", resp)
	}
}

func TestGuideDdnstoAddressServiceMapsLegacyErrorWording(t *testing.T) {
	t.Parallel()

	writer := &fakeGuideDDNSWriter{
		ddnstoAddressStderr: " boom",
		ddnstoAddressErr:    errors.New("address failed"),
	}
	service := GuideDdnstoAddressService{writer: writer}

	if _, err := service.UpdateAddress(context.Background(), GuideDdnstoAddressInput{Address: "https://demo.example.com"}); err == nil || err.Error() != "ddnsto地址信息保存失败 boom" {
		t.Fatalf("unexpected address error: %v", err)
	}
}

func TestGuideDDNSServiceReturnsPendingChangeResponse(t *testing.T) {
	t.Parallel()

	service := GuideDDNSService{
		reader: &fakeGuideDDNSReader{pending: true},
		writer: &fakeGuideDDNSWriter{},
	}

	resp, err := service.Update(context.Background(), GuideDDNSInput{SessionID: "sess-1"})
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if resp == nil || resp.Error != models.ResponseError("-100") || resp.Scope != models.ResponseScope("guide.ddns") {
		t.Fatalf("unexpected pending response: %#v", resp)
	}
}

func TestGuideDDNSServiceBuildsIPv4NetworkConfig(t *testing.T) {
	t.Parallel()

	writer := &fakeGuideDDNSWriter{}
	service := GuideDDNSService{
		reader: &fakeGuideDDNSReader{
			outbound: &GuideDDNSOutboundSnapshot{
				IPv4: &GuideDDNSInterfaceSnapshot{InterfaceName: "wan", IP: "1.2.3.4"},
			},
			publicIPv4: true,
		},
		writer: writer,
	}

	resp, err := service.Update(context.Background(), GuideDDNSInput{
		SessionID:   "sess-1",
		Domain:      "demo.example.com",
		IPVersion:   "ipv4",
		Password:    " pass ",
		ServiceName: "ali",
		UserName:    " user ",
	})
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("unexpected success response: %#v", resp)
	}
	if writer.startConfigName != "myddns_ipv4" {
		t.Fatalf("unexpected start config name: %q", writer.startConfigName)
	}
	expected := []string{
		"uci set ddns.myddns_ipv4=service",
		"uci set ddns.myddns_ipv4.enabled='1'",
		"uci set ddns.myddns_ipv4.use_ipv6=0",
		"uci set ddns.myddns_ipv4.service_name=aliyun.com",
		"uci set ddns.myddns_ipv4.lookup_host=demo.example.com",
		"uci set ddns.myddns_ipv4.domain=demo.example.com",
		"uci set ddns.myddns_ipv4.username=user",
		"uci set ddns.myddns_ipv4.password=pass",
		"uci set ddns.myddns_ipv4.interface=wan",
		"uci set ddns.myddns_ipv4.use_syslog=2",
		"uci set ddns.myddns_ipv4.check_unit=minutes",
		"uci set ddns.myddns_ipv4.force_unit=minutes",
		"uci set ddns.myddns_ipv4.retry_unit=seconds",
		"uci set ddns.myddns_ipv4.ip_source=network",
		"uci set ddns.myddns_ipv4.ip_network=wan",
		"uci commit ddns",
	}
	if len(writer.ddnsCmds) != len(expected) {
		t.Fatalf("unexpected cmd count: %#v", writer.ddnsCmds)
	}
	for i := range expected {
		if writer.ddnsCmds[i] != expected[i] {
			t.Fatalf("unexpected cmd[%d]: %q", i, writer.ddnsCmds[i])
		}
	}
}

func TestGuideDDNSServiceBuildsIPv4WebConfigWithoutPublicNet(t *testing.T) {
	t.Parallel()

	writer := &fakeGuideDDNSWriter{}
	service := GuideDDNSService{
		reader: &fakeGuideDDNSReader{
			outbound: &GuideDDNSOutboundSnapshot{
				IPv4: &GuideDDNSInterfaceSnapshot{InterfaceName: "wan", IP: "10.0.0.2"},
			},
			publicIPv4: false,
		},
		writer: writer,
	}

	if _, err := service.Update(context.Background(), GuideDDNSInput{
		Domain:      "demo.example.com",
		IPVersion:   "ipv4",
		Password:    "pass",
		ServiceName: "oray",
		UserName:    "user",
	}); err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	foundURL := false
	foundDelete := false
	for _, cmd := range writer.ddnsCmds {
		if cmd == "uci set ddns.myddns_ipv4.ip_url=4.ipw.cn" {
			foundURL = true
		}
		if cmd == "uci del ddns.myddns_ipv4.ip_network" {
			foundDelete = true
		}
	}
	if !foundURL || !foundDelete {
		t.Fatalf("unexpected web config cmds: %#v", writer.ddnsCmds)
	}
}

func TestGuideDDNSServiceMapsInvalidServiceName(t *testing.T) {
	t.Parallel()

	service := GuideDDNSService{
		reader: &fakeGuideDDNSReader{
			outbound: &GuideDDNSOutboundSnapshot{
				IPv4: &GuideDDNSInterfaceSnapshot{InterfaceName: "wan", IP: "1.2.3.4"},
			},
			publicIPv4: true,
		},
		writer: &fakeGuideDDNSWriter{},
	}

	if _, err := service.Update(context.Background(), GuideDDNSInput{IPVersion: "ipv4", ServiceName: "foo"}); err == nil || err.Error() != "serviceName参数错误foo" {
		t.Fatalf("unexpected service-name error: %v", err)
	}
}

func TestGuideDDNSServiceMapsApplyFailure(t *testing.T) {
	t.Parallel()

	service := GuideDDNSService{
		reader: &fakeGuideDDNSReader{
			outbound: &GuideDDNSOutboundSnapshot{
				IPv4: &GuideDDNSInterfaceSnapshot{InterfaceName: "wan", IP: "1.2.3.4"},
			},
			publicIPv4: true,
		},
		writer: &fakeGuideDDNSWriter{ddnsErr: errors.New("apply failed")},
	}

	if _, err := service.Update(context.Background(), GuideDDNSInput{
		Domain:      "demo.example.com",
		IPVersion:   "ipv4",
		Password:    "pass",
		ServiceName: "ali",
		UserName:    "user",
	}); err == nil || err.Error() != "修改ddns配置失败" {
		t.Fatalf("unexpected apply error: %v", err)
	}
}

func TestGuideDDNSServiceIgnoresStartFailure(t *testing.T) {
	t.Parallel()

	writer := &fakeGuideDDNSWriter{startErr: errors.New("start failed")}
	service := GuideDDNSService{
		reader: &fakeGuideDDNSReader{
			outbound: &GuideDDNSOutboundSnapshot{
				IPv6: &GuideDDNSInterfaceSnapshot{InterfaceName: "wan6", IP: "fd00::2"},
			},
			publicIPv6: false,
		},
		writer: writer,
	}

	resp, err := service.Update(context.Background(), GuideDDNSInput{
		Domain:      "demo.example.com",
		IPVersion:   "ipv6",
		Password:    "pass",
		ServiceName: "dnspod",
		UserName:    "user",
	})
	if err != nil {
		t.Fatalf("unexpected start error propagation: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("unexpected success response: %#v", resp)
	}
	if writer.startConfigName != "myddns_ipv6" {
		t.Fatalf("unexpected ipv6 start config: %q", writer.startConfigName)
	}
}

func TestServiceBackendPostGuideDdnstoCompatibility(t *testing.T) {
	prev := guideDdnstoEnable
	defer func() { guideDdnstoEnable = prev }()

	expected := &models.SDKNormalResponse{Success: func() *models.ResponseSuccess {
		success := models.ResponseSuccess(0)
		return &success
	}()}
	captured := GuideDdnstoEnableInput{}
	guideDdnstoEnable = func(ctx context.Context, input GuideDdnstoEnableInput) (*models.SDKNormalResponse, error) {
		captured = input
		return expected, nil
	}

	resp, err := (&ServiceBackend{}).PostGuideDdnsto(context.Background(), reqWithBody(t, `{"token":"token-abc"}`))
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if captured.Token != "token-abc" {
		t.Fatalf("unexpected enable input: %#v", captured)
	}
	if resp != expected {
		t.Fatalf("unexpected response passthrough: %#v", resp)
	}
}

func TestServiceBackendPostGuideDdnstoAddressCompatibility(t *testing.T) {
	prev := guideDdnstoAddress
	defer func() { guideDdnstoAddress = prev }()

	expected := &models.SDKNormalResponse{Success: func() *models.ResponseSuccess {
		success := models.ResponseSuccess(0)
		return &success
	}()}
	captured := GuideDdnstoAddressInput{}
	guideDdnstoAddress = func(ctx context.Context, input GuideDdnstoAddressInput) (*models.SDKNormalResponse, error) {
		captured = input
		return expected, nil
	}

	resp, err := (&ServiceBackend{}).PostGuideDdnstoAddress(context.Background(), reqWithBody(t, `{"address":"https://demo.example.com"}`))
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if captured.Address != "https://demo.example.com" {
		t.Fatalf("unexpected address input: %#v", captured)
	}
	if resp != expected {
		t.Fatalf("unexpected response passthrough: %#v", resp)
	}
}

func TestServiceBackendPostGuideDdnsCompatibility(t *testing.T) {
	prev := guideDDNSUpdate
	defer func() { guideDDNSUpdate = prev }()

	expected := &models.SDKNormalResponse{Success: func() *models.ResponseSuccess {
		success := models.ResponseSuccess(0)
		return &success
	}()}
	captured := GuideDDNSInput{}
	guideDDNSUpdate = func(ctx context.Context, input GuideDDNSInput) (*models.SDKNormalResponse, error) {
		captured = input
		return expected, nil
	}

	req := reqWithBody(t, `{"domain":"demo.example.com","ipVersion":"ipv4","password":"pass","serviceName":"ali","userName":"user"}`)
	req.AddCookie(&http.Cookie{Name: "sysauth", Value: "sess-1"})
	resp, err := (&ServiceBackend{}).PostGuideDdns(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if captured.SessionID != "sess-1" || captured.Domain != "demo.example.com" || captured.IPVersion != "ipv4" || captured.Password != "pass" || captured.ServiceName != "ali" || captured.UserName != "user" {
		t.Fatalf("unexpected ddns input: %#v", captured)
	}
	if resp != expected {
		t.Fatalf("unexpected response passthrough: %#v", resp)
	}
}

func TestServiceBackendPostGuideDdnsMapsRequestParseError(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/guide/ddns", strings.NewReader("{"))
	if err != nil {
		t.Fatalf("unexpected request build error: %v", err)
	}
	if _, err := (&ServiceBackend{}).PostGuideDdns(context.Background(), req); err == nil || err.Error() != "请求解析失败" {
		t.Fatalf("unexpected parse error: %v", err)
	}
}
