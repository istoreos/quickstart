package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestDefaultGuideTransparentGatewayWriterSetDHCPEnabledPreservesLegacyDefaults(t *testing.T) {
	originalBatch := writeGuideTransparentGatewayBatchRun
	defer func() { writeGuideTransparentGatewayBatchRun = originalBatch }()

	var got []string
	writeGuideTransparentGatewayBatchRun = func(ctx context.Context, cmdList []string) error {
		got = append([]string(nil), cmdList...)
		return nil
	}

	writer := newDefaultGuideTransparentGatewayWriter()
	if err := writer.SetDHCP(ctxbg, true); err != nil {
		t.Fatalf("unexpected DHCP enable error: %v", err)
	}

	want := []string{
		"uci -q del dhcp.lan.ignore || true",
		"uci set dhcp.lan.dhcpv4=server",
		"uci set dhcp.lan.start=100",
		"uci set dhcp.lan.limit=150",
		"uci set dhcp.lan.leasetime=12h",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected DHCP enable commands:\nwant=%#v\ngot=%#v", want, got)
	}
}

func TestDefaultGuideTransparentGatewayWriterSetDHCPDisabledPreservesLegacyIntent(t *testing.T) {
	originalBatch := writeGuideTransparentGatewayBatchRun
	defer func() { writeGuideTransparentGatewayBatchRun = originalBatch }()

	var got []string
	writeGuideTransparentGatewayBatchRun = func(ctx context.Context, cmdList []string) error {
		got = append([]string(nil), cmdList...)
		return nil
	}

	writer := newDefaultGuideTransparentGatewayWriter()
	if err := writer.SetDHCP(ctxbg, false); err != nil {
		t.Fatalf("unexpected DHCP disable error: %v", err)
	}

	want := []string{"uci set dhcp.lan.ignore=1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected DHCP disable commands:\nwant=%#v\ngot=%#v", want, got)
	}
}

func TestDefaultGuideTransparentGatewayWriterSetLan6PreservesLegacyBranches(t *testing.T) {
	originalBatch := writeGuideTransparentGatewayBatchRun
	defer func() { writeGuideTransparentGatewayBatchRun = originalBatch }()

	var calls [][]string
	writeGuideTransparentGatewayBatchRun = func(ctx context.Context, cmdList []string) error {
		calls = append(calls, append([]string(nil), cmdList...))
		return nil
	}

	writer := newDefaultGuideTransparentGatewayWriter()
	if err := writer.SetLan6(ctxbg, true); err != nil {
		t.Fatalf("unexpected lan6 enable error: %v", err)
	}
	if err := writer.SetLan6(ctxbg, false); err != nil {
		t.Fatalf("unexpected lan6 disable error: %v", err)
	}

	if len(calls) != 2 {
		t.Fatalf("expected 2 lan6 calls, got %d", len(calls))
	}

	enableWant := []string{
		"uci -q batch <<-EOF >/dev/null",
		"set network.lan6=interface",
		"set network.lan6.proto=dhcpv6",
		"set network.lan6.device=@lan",
		"EOF",
		"",
	}
	disableWant := []string{"uci -q delete network.lan6 || true"}
	if !reflect.DeepEqual(calls[0], enableWant) {
		t.Fatalf("unexpected lan6 enable commands:\nwant=%#v\ngot=%#v", enableWant, calls[0])
	}
	if !reflect.DeepEqual(calls[1], disableWant) {
		t.Fatalf("unexpected lan6 disable commands:\nwant=%#v\ngot=%#v", disableWant, calls[1])
	}
}

func TestBuildGuideTransparentGatewayPending(t *testing.T) {
	withFirewall := buildGuideTransparentGatewayPending(models.GuideGatewayRouterRequest{EnableNat: true}, true)
	withoutFirewall := buildGuideTransparentGatewayPending(models.GuideGatewayRouterRequest{EnableNat: false}, false)

	if !reflect.DeepEqual(withFirewall, []string{"firewall", "dhcp", "network"}) {
		t.Fatalf("unexpected pending with firewall: %v", withFirewall)
	}
	if !reflect.DeepEqual(withoutFirewall, []string{"dhcp", "network"}) {
		t.Fatalf("unexpected pending without firewall: %v", withoutFirewall)
	}
}

var ctxbg = context.Background()
