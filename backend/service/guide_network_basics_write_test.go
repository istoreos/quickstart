package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestDefaultGuideNetworkBasicsWriterSetDNSConfigPreservesAutoAndManualSemantics(t *testing.T) {
	originalSetDNS := writeGuideNetworkBasicsDNS
	defer func() {
		writeGuideNetworkBasicsDNS = originalSetDNS
	}()

	var gotInterface string
	var gotProto string
	var gotIPs []string
	writeGuideNetworkBasicsDNS = func(ctx context.Context, it string, dnsProto string, dnsIPs []string) error {
		gotInterface = it
		gotProto = dnsProto
		gotIPs = append([]string(nil), dnsIPs...)
		return nil
	}

	writer := newDefaultGuideNetworkBasicsWriter()
	if err := writer.SetDNSConfig(context.Background(), GuideSetDNSConfigInput{
		InterfaceName: "wan",
		DNSProto:      "auto",
	}); err != nil {
		t.Fatalf("unexpected auto DNS write error: %v", err)
	}
	if gotInterface != "wan" || gotProto != "auto" || len(gotIPs) != 0 {
		t.Fatalf("unexpected auto DNS write args: interface=%q proto=%q ips=%v", gotInterface, gotProto, gotIPs)
	}

	if err := writer.SetDNSConfig(context.Background(), GuideSetDNSConfigInput{
		InterfaceName: "wan",
		DNSProto:      "manual",
		ManualDNSIP:   []string{"1.1.1.1", "8.8.8.8"},
	}); err != nil {
		t.Fatalf("unexpected manual DNS write error: %v", err)
	}
	if gotInterface != "wan" || gotProto != "manual" || !reflect.DeepEqual(gotIPs, []string{"1.1.1.1", "8.8.8.8"}) {
		t.Fatalf("unexpected manual DNS write args: interface=%q proto=%q ips=%v", gotInterface, gotProto, gotIPs)
	}
}

func TestDefaultGuideNetworkBasicsWriterSetWANModePreservesLegacyHelpers(t *testing.T) {
	originalInterface := writeGuideNetworkBasicsInterface
	originalPPPoE := writeGuideNetworkBasicsPPPoE
	defer func() {
		writeGuideNetworkBasicsInterface = originalInterface
		writeGuideNetworkBasicsPPPoE = originalPPPoE
	}()

	var interfaceInput GuideSetWANInterfaceInput
	var pppoeInput GuideSetPPPoEInput
	writeGuideNetworkBasicsInterface = func(ctx context.Context, input GuideSetWANInterfaceInput) error {
		interfaceInput = input
		return nil
	}
	writeGuideNetworkBasicsPPPoE = func(ctx context.Context, input GuideSetPPPoEInput) error {
		pppoeInput = input
		return nil
	}

	writer := newDefaultGuideNetworkBasicsWriter()
	if err := writer.SetWANInterfaceMode(context.Background(), GuideSetWANInterfaceInput{
		InterfaceName: "wan",
		WanProto:      "static",
		StaticIP:      "10.0.0.2",
		SubnetMask:    "255.255.255.0",
		Gateway:       "10.0.0.1",
	}); err != nil {
		t.Fatalf("unexpected WAN interface write error: %v", err)
	}
	if interfaceInput.InterfaceName != "wan" || interfaceInput.WanProto != "static" || interfaceInput.StaticIP != "10.0.0.2" || interfaceInput.SubnetMask != "255.255.255.0" || interfaceInput.Gateway != "10.0.0.1" {
		t.Fatalf("unexpected WAN interface input: %#v", interfaceInput)
	}

	if err := writer.SetPPPoE(context.Background(), GuideSetPPPoEInput{
		Account:  "user",
		Password: "pw",
	}); err != nil {
		t.Fatalf("unexpected PPPOE write error: %v", err)
	}
	if pppoeInput.Account != "user" || pppoeInput.Password != "pw" {
		t.Fatalf("unexpected PPPOE input: %#v", pppoeInput)
	}
}

func TestDefaultGuideNetworkBasicsWriterSetLANConfigPreservesDhcpAndMasqSemantics(t *testing.T) {
	originalLAN := writeGuideNetworkBasicsLAN
	originalEnableDHCP := writeGuideNetworkBasicsEnableLanDHCP
	originalSetMasq := writeGuideNetworkBasicsSetLanMasq
	originalDHCPRange := writeGuideNetworkBasicsLanDHCPRange
	defer func() {
		writeGuideNetworkBasicsLAN = originalLAN
		writeGuideNetworkBasicsEnableLanDHCP = originalEnableDHCP
		writeGuideNetworkBasicsSetLanMasq = originalSetMasq
		writeGuideNetworkBasicsLanDHCPRange = originalDHCPRange
	}()

	var lanInput GuideSetLANConfigInput
	var enableCalls int
	var masqValues []bool
	var gotDhcpStart int
	var gotDhcpLimit int
	writeGuideNetworkBasicsLAN = func(input GuideSetLANConfigInput) error {
		lanInput = input
		return nil
	}
	writeGuideNetworkBasicsLanDHCPRange = func(ctx context.Context, start int, limit int) error {
		gotDhcpStart = start
		gotDhcpLimit = limit
		return nil
	}
	writeGuideNetworkBasicsEnableLanDHCP = func(ctx context.Context) {
		enableCalls++
	}
	writeGuideNetworkBasicsSetLanMasq = func(ctx context.Context, enable bool) bool {
		masqValues = append(masqValues, enable)
		return true
	}

	writer := newDefaultGuideNetworkBasicsWriter()
	pending, err := writer.SetLANConfig(context.Background(), GuideSetLANConfigInput{
		LanIP:      "192.168.100.1",
		NetMask:    "255.255.255.0",
		EnableDhcp: true,
		DhcpStart:  "192.168.100.100",
		DhcpEnd:    "192.168.100.249",
	})
	if err != nil {
		t.Fatalf("unexpected LAN config write error: %v", err)
	}
	if lanInput.LanIP != "192.168.100.1" || lanInput.NetMask != "255.255.255.0" || !lanInput.EnableDhcp {
		t.Fatalf("unexpected LAN input: %#v", lanInput)
	}
	if gotDhcpStart != 100 || gotDhcpLimit != 149 {
		t.Fatalf("unexpected DHCP range args: start=%d limit=%d", gotDhcpStart, gotDhcpLimit)
	}
	if enableCalls != 1 {
		t.Fatalf("expected enable LAN DHCP once, got %d", enableCalls)
	}
	if len(masqValues) != 0 {
		t.Fatalf("unexpected masq calls: %v", masqValues)
	}
	if !reflect.DeepEqual(pending, []string{"dhcp", "network"}) {
		t.Fatalf("unexpected pending configs: %v", pending)
	}
}

func TestWriteGuideNetworkBasicsDeleteLan6IsIdempotent(t *testing.T) {
	originalBatch := writeGuideNetworkBasicsBatchRun
	defer func() { writeGuideNetworkBasicsBatchRun = originalBatch }()

	var got []string
	writeGuideNetworkBasicsBatchRun = func(ctx context.Context, cmdList []string) error {
		got = append([]string(nil), cmdList...)
		return nil
	}

	if err := writeGuideNetworkBasicsDeleteLan6(context.Background()); err != nil {
		t.Fatalf("unexpected lan6 delete error: %v", err)
	}

	want := []string{"uci -q delete network.lan6 || true"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lan6 delete command: got=%#v want=%#v", got, want)
	}
}

func TestDefaultGuideNetworkBasicsApplyDelegatesPendingConfigs(t *testing.T) {
	originalApply := applyGuideNetworkBasicsPending
	defer func() {
		applyGuideNetworkBasicsPending = originalApply
	}()

	var gotPending []string
	applyGuideNetworkBasicsPending = func(ctx context.Context, pending []string) error {
		gotPending = append([]string(nil), pending...)
		return nil
	}

	apply := newDefaultGuideNetworkBasicsApply()
	if err := apply.Apply(context.Background(), []string{"firewall", "dhcp", "network"}); err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}
	if !reflect.DeepEqual(gotPending, []string{"firewall", "dhcp", "network"}) {
		t.Fatalf("unexpected pending apply list: %v", gotPending)
	}
}

func TestDefaultGuideNetworkBasicsWriterAndApplyPropagateErrors(t *testing.T) {
	originalSetDNS := writeGuideNetworkBasicsDNS
	originalApply := applyGuideNetworkBasicsPending
	defer func() {
		writeGuideNetworkBasicsDNS = originalSetDNS
		applyGuideNetworkBasicsPending = originalApply
	}()

	expectedErr := errors.New("write failed")
	writeGuideNetworkBasicsDNS = func(ctx context.Context, it string, dnsProto string, dnsIPs []string) error {
		return expectedErr
	}
	writer := newDefaultGuideNetworkBasicsWriter()
	if err := writer.SetDNSConfig(context.Background(), GuideSetDNSConfigInput{
		InterfaceName: "wan",
		DNSProto:      "auto",
	}); !errors.Is(err, expectedErr) {
		t.Fatalf("expected DNS write error, got %v", err)
	}

	applyErr := errors.New("apply failed")
	applyGuideNetworkBasicsPending = func(ctx context.Context, pending []string) error {
		return applyErr
	}
	apply := newDefaultGuideNetworkBasicsApply()
	if err := apply.Apply(context.Background(), []string{"network"}); !errors.Is(err, applyErr) {
		t.Fatalf("expected apply error, got %v", err)
	}
}

func TestBuildGuideNetworkBasicsPendingForWANMode(t *testing.T) {
	pending := buildGuideNetworkBasicsPendingForWANMode(models.GuideClientModeRequest{
		EnableLanDhcp: true,
	}, true)
	if !reflect.DeepEqual(pending, []string{"firewall", "dhcp", "network"}) {
		t.Fatalf("unexpected WAN-mode pending configs: %v", pending)
	}

	pending = buildGuideNetworkBasicsPendingForWANMode(models.GuidePppoeRequest{
		EnableLanDhcp: false,
	}, false)
	if !reflect.DeepEqual(pending, []string{"network"}) {
		t.Fatalf("unexpected PPPOE pending configs: %v", pending)
	}
}
