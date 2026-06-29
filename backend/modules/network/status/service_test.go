package status

import (
	"context"
	"errors"
	"testing"
)

type fakeReader struct {
	snapshot  Snapshot
	dnsConfig DNSConfig
	err       error
}

func (reader *fakeReader) Read(ctx context.Context) (Snapshot, DNSConfig, error) {
	return reader.snapshot, reader.dnsConfig, reader.err
}

type fakeChecker struct {
	status      OnlineStatus
	err         error
	lastIP      string
	lastGateway string
	lastDNS     []string
}

func (checker *fakeChecker) GetStatus(ip string, gateway string, dns []string) (OnlineStatus, error) {
	checker.lastIP = ip
	checker.lastGateway = gateway
	checker.lastDNS = append([]string(nil), dns...)
	return checker.status, checker.err
}

type fakeMarker struct {
	called int
}

func (marker *fakeMarker) MarkSetupFinish(ctx context.Context) {
	marker.called++
}

func TestServiceWithoutCheckerBuildsStatusResult(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeReader{
		snapshot: Snapshot{
			IPv4: &IPv4Snapshot{
				Address:       "10.0.0.2",
				Mask:          24,
				Proto:         "dhcp",
				Gateway:       "10.0.0.1",
				UptimeSeconds: 90,
			},
			IPv6Addr:       "fe80::2",
			ResolvedIfName: "wan",
		},
		dnsConfig: DNSConfig{Proto: "auto", DNSList: []string{"1.1.1.1"}},
	}, nil, nil)

	resp, err := svc.GetNetworkStatus(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result")
	}
	if resp.Result.NetworkInfo != "" {
		t.Fatalf("expected empty network info without checker, got %q", resp.Result.NetworkInfo)
	}
	if resp.Result.Ipv4addr != "10.0.0.2" || resp.Result.Gateway != "10.0.0.1" {
		t.Fatalf("unexpected result mapping: %#v", resp.Result)
	}
	if resp.Result.UptimeStamp != 90 || resp.Result.Uptime == "" {
		t.Fatalf("expected uptime mapped, got %#v", resp.Result)
	}
}

func TestServiceWithCheckerDoesNotMarkWhenSetupFinishIsFalse(t *testing.T) {
	t.Parallel()

	checker := &fakeChecker{status: OnlineOK}
	marker := &fakeMarker{}
	svc := NewService(&fakeReader{
		snapshot: Snapshot{
			IPv4:           &IPv4Snapshot{Address: "10.0.0.2", Mask: 24, Proto: "dhcp", Gateway: "10.0.0.1"},
			ResolvedIfName: "wan",
		},
		dnsConfig: DNSConfig{Proto: "auto", DNSList: []string{"1.1.1.1"}},
	}, checker, marker)

	resp, err := svc.GetNetworkStatus(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if resp.Result.NetworkInfo != OnlineOK.String() {
		t.Fatalf("expected network info %q, got %q", OnlineOK.String(), resp.Result.NetworkInfo)
	}
	if checker.lastIP != "10.0.0.2" || checker.lastGateway != "10.0.0.1" {
		t.Fatalf("checker did not receive mapped network fields: %#v", checker)
	}
	if marker.called != 0 {
		t.Fatalf("expected marker not called when setupFinish=false, got %d", marker.called)
	}
}

func TestServiceMarksSetupFinishOnlyForCompatibleStates(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		status     OnlineStatus
		wantMarked int
	}{
		{name: "detecting", status: OnlineDetecting, wantMarked: 0},
		{name: "offline", status: OnlineFailedOffline, wantMarked: 0},
		{name: "dns-failed", status: OnlineFailedDNS, wantMarked: 1},
		{name: "ok", status: OnlineOK, wantMarked: 1},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			marker := &fakeMarker{}
			svc := NewService(&fakeReader{
				snapshot: Snapshot{
					IPv4:           &IPv4Snapshot{Address: "10.0.0.2", Gateway: "10.0.0.1"},
					ResolvedIfName: "wan",
				},
				dnsConfig: DNSConfig{Proto: "auto", DNSList: []string{"1.1.1.1"}},
			}, &fakeChecker{status: tc.status}, marker)

			resp, err := svc.GetNetworkStatus(context.Background(), true)
			if err != nil {
				t.Fatalf("unexpected service error: %v", err)
			}
			if resp.Result.NetworkInfo != tc.status.String() {
				t.Fatalf("expected network info %q, got %q", tc.status.String(), resp.Result.NetworkInfo)
			}
			if marker.called != tc.wantMarked {
				t.Fatalf("expected marker called %d times, got %d", tc.wantMarked, marker.called)
			}
		})
	}
}

func TestServicePropagatesReaderAndCheckerErrors(t *testing.T) {
	t.Parallel()

	readerErr := errors.New("reader failed")
	svc := NewService(&fakeReader{err: readerErr}, nil, nil)
	if _, err := svc.GetNetworkStatus(context.Background(), false); !errors.Is(err, readerErr) {
		t.Fatalf("expected reader error, got %v", err)
	}

	checkErr := errors.New("checker failed")
	svc = NewService(&fakeReader{
		snapshot: Snapshot{
			IPv4:           &IPv4Snapshot{Address: "10.0.0.2", Gateway: "10.0.0.1"},
			ResolvedIfName: "wan",
		},
		dnsConfig: DNSConfig{Proto: "auto", DNSList: []string{"1.1.1.1"}},
	}, &fakeChecker{err: checkErr}, nil)
	if _, err := svc.GetNetworkStatus(context.Background(), true); !errors.Is(err, checkErr) {
		t.Fatalf("expected checker error, got %v", err)
	}
}
