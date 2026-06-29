package publicaddress

import "testing"

func TestSelectNetworkPublicAddress(t *testing.T) {
	t.Parallel()

	snapshot := Snapshot{
		IPv4: "203.0.113.10",
		IPv6: "2001:db8::1",
	}

	ipv4, err := selectNetworkPublicAddress(snapshot, "ipv4")
	if err != nil {
		t.Fatalf("unexpected ipv4 selection error: %v", err)
	}
	if ipv4 != "203.0.113.10" {
		t.Fatalf("expected ipv4 address, got %q", ipv4)
	}

	ipv6, err := selectNetworkPublicAddress(snapshot, "ipv6")
	if err != nil {
		t.Fatalf("unexpected ipv6 selection error: %v", err)
	}
	if ipv6 != "2001:db8::1" {
		t.Fatalf("expected ipv6 address, got %q", ipv6)
	}
}

func TestSelectNetworkPublicAddressRejectsInvalidVersion(t *testing.T) {
	t.Parallel()

	_, err := selectNetworkPublicAddress(Snapshot{}, "ipv10")
	if err == nil || err.Error() != "IPVersion参数错误ipv10" {
		t.Fatalf("expected invalid version error, got %v", err)
	}
}

func TestSelectNetworkPublicAddressRejectsMissingVersionAddress(t *testing.T) {
	t.Parallel()

	_, err := selectNetworkPublicAddress(Snapshot{}, "ipv4")
	if err == nil || err.Error() != "没有获取到ipv4信息" {
		t.Fatalf("expected missing ipv4 error, got %v", err)
	}

	_, err = selectNetworkPublicAddress(Snapshot{}, "ipv6")
	if err == nil || err.Error() != "没有获取到ipv6信息" {
		t.Fatalf("expected missing ipv6 error, got %v", err)
	}
}

func TestBuildNetworkPublicAddressResult(t *testing.T) {
	t.Parallel()

	pub := buildNetworkPublicAddressResult("203.0.113.10", true)
	if pub.Result == nil || pub.Result.Address != "203.0.113.10" {
		t.Fatalf("expected public address in response, got %#v", pub)
	}

	private := buildNetworkPublicAddressResult("192.168.1.2", false)
	if private.Result == nil {
		t.Fatalf("expected result object, got %#v", private)
	}
	if private.Result.Address != "" {
		t.Fatalf("expected private address to map to empty result, got %q", private.Result.Address)
	}
}
