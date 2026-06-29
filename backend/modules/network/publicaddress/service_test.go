package publicaddress

import (
	"errors"
	"testing"
)

type fakeNetworkPublicAddressReader struct {
	snapshot Snapshot
	err      error
}

func (reader *fakeNetworkPublicAddressReader) Read() (Snapshot, error) {
	return reader.snapshot, reader.err
}

type fakeNetworkPublicAddressClassifier struct {
	results map[string]bool
	calls   []string
}

func (classifier *fakeNetworkPublicAddressClassifier) IsPublic(ipVersion string, address string) bool {
	classifier.calls = append(classifier.calls, ipVersion+":"+address)
	return classifier.results[ipVersion+":"+address]
}

func TestNetworkPublicAddressServiceIPv4Public(t *testing.T) {
	t.Parallel()

	classifier := &fakeNetworkPublicAddressClassifier{
		results: map[string]bool{"ipv4:203.0.113.10": true},
	}
	svc := NewService(&fakeNetworkPublicAddressReader{snapshot: Snapshot{IPv4: "203.0.113.10"}}, classifier)

	resp, err := svc.CheckPublicAddress("ipv4")
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if resp.Result == nil || resp.Result.Address != "203.0.113.10" {
		t.Fatalf("expected public ipv4 address in response, got %#v", resp)
	}
	if len(classifier.calls) != 1 || classifier.calls[0] != "ipv4:203.0.113.10" {
		t.Fatalf("unexpected classifier calls: %#v", classifier.calls)
	}
}

func TestNetworkPublicAddressServicePrivateAddressReturnsEmpty(t *testing.T) {
	t.Parallel()

	classifier := &fakeNetworkPublicAddressClassifier{
		results: map[string]bool{"ipv4:192.168.1.2": false},
	}
	svc := NewService(&fakeNetworkPublicAddressReader{snapshot: Snapshot{IPv4: "192.168.1.2"}}, classifier)

	resp, err := svc.CheckPublicAddress("ipv4")
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if resp.Result == nil || resp.Result.Address != "" {
		t.Fatalf("expected empty private-address response, got %#v", resp)
	}
}

func TestNetworkPublicAddressServiceIPv6Public(t *testing.T) {
	t.Parallel()

	classifier := &fakeNetworkPublicAddressClassifier{
		results: map[string]bool{"ipv6:2001:db8::1": true},
	}
	svc := NewService(&fakeNetworkPublicAddressReader{snapshot: Snapshot{IPv6: "2001:db8::1"}}, classifier)

	resp, err := svc.CheckPublicAddress("ipv6")
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if resp.Result == nil || resp.Result.Address != "2001:db8::1" {
		t.Fatalf("expected public ipv6 address in response, got %#v", resp)
	}
}

func TestNetworkPublicAddressServicePropagatesSelectionAndReaderErrors(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeNetworkPublicAddressReader{snapshot: Snapshot{}}, &fakeNetworkPublicAddressClassifier{})
	if _, err := svc.CheckPublicAddress("ipv10"); err == nil || err.Error() != "IPVersion参数错误ipv10" {
		t.Fatalf("expected invalid version error, got %v", err)
	}

	readErr := errors.New("reader failed")
	svc = NewService(&fakeNetworkPublicAddressReader{err: readErr}, &fakeNetworkPublicAddressClassifier{})
	if _, err := svc.CheckPublicAddress("ipv4"); !errors.Is(err, readErr) {
		t.Fatalf("expected reader error, got %v", err)
	}
}
