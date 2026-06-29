package service

import (
	"errors"
	"sync"
	"testing"
)

var networkPublicAddressStoreTestMu sync.Mutex

func TestDefaultNetworkPublicAddressReader(t *testing.T) {
	networkPublicAddressStoreTestMu.Lock()
	defer networkPublicAddressStoreTestMu.Unlock()

	oldOutbound := networkPublicAddressOutboundInterfaces
	defer func() {
		networkPublicAddressOutboundInterfaces = oldOutbound
	}()

	networkPublicAddressOutboundInterfaces = func() (*DefaultInterfaces, error) {
		return &DefaultInterfaces{
			ipv4: &DefaultInterface{ip: "203.0.113.10"},
			ipv6: &DefaultInterface{ip: "2001:db8::1"},
		}, nil
	}

	snapshot, err := newDefaultNetworkPublicAddressReader().Read()
	if err != nil {
		t.Fatalf("unexpected reader error: %v", err)
	}
	if snapshot.IPv4 != "203.0.113.10" || snapshot.IPv6 != "2001:db8::1" {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
}

func TestDefaultNetworkPublicAddressReaderPropagatesOutboundError(t *testing.T) {
	networkPublicAddressStoreTestMu.Lock()
	defer networkPublicAddressStoreTestMu.Unlock()

	oldOutbound := networkPublicAddressOutboundInterfaces
	defer func() {
		networkPublicAddressOutboundInterfaces = oldOutbound
	}()

	wantErr := errors.New("outbound failed")
	networkPublicAddressOutboundInterfaces = func() (*DefaultInterfaces, error) {
		return nil, wantErr
	}

	_, err := newDefaultNetworkPublicAddressReader().Read()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected outbound error, got %v", err)
	}
}

func TestDefaultNetworkPublicAddressClassifier(t *testing.T) {
	networkPublicAddressStoreTestMu.Lock()
	defer networkPublicAddressStoreTestMu.Unlock()

	oldIPv4 := networkPublicAddressIsPublicIPv4
	oldIPv6 := networkPublicAddressIsPublicIPv6
	defer func() {
		networkPublicAddressIsPublicIPv4 = oldIPv4
		networkPublicAddressIsPublicIPv6 = oldIPv6
	}()

	var gotIPv4 string
	var gotIPv6 string
	networkPublicAddressIsPublicIPv4 = func(addr string) bool {
		gotIPv4 = addr
		return true
	}
	networkPublicAddressIsPublicIPv6 = func(addr string) bool {
		gotIPv6 = addr
		return false
	}

	classifier := newDefaultNetworkPublicAddressClassifier()
	if !classifier.IsPublic("ipv4", "203.0.113.10") {
		t.Fatal("expected ipv4 classifier result true")
	}
	if gotIPv4 != "203.0.113.10" {
		t.Fatalf("expected ipv4 classifier to receive address, got %q", gotIPv4)
	}
	if classifier.IsPublic("ipv6", "2001:db8::1") {
		t.Fatal("expected ipv6 classifier result false")
	}
	if gotIPv6 != "2001:db8::1" {
		t.Fatalf("expected ipv6 classifier to receive address, got %q", gotIPv6)
	}
}
