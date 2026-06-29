package portlist

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStatusReader struct {
	ports []*models.NetworkPortInfo
	err   error
}

func (reader *fakeStatusReader) Read(ctx context.Context) ([]*models.NetworkPortInfo, error) {
	return reader.ports, reader.err
}

type fakeMembershipReader struct {
	memberships []MembershipSnapshot
	err         error
}

func (reader *fakeMembershipReader) Read(ctx context.Context) ([]MembershipSnapshot, error) {
	return reader.memberships, reader.err
}

func TestServiceMergesMembershipIntoPorts(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStatusReader{
		ports: []*models.NetworkPortInfo{
			{Name: "eth0", InterfaceNames: []string{"existing"}},
			{Name: "lan1", Master: "br-lan"},
		},
	}, &fakeMembershipReader{
		memberships: []MembershipSnapshot{
			{InterfaceName: "wan", Device: "eth0"},
			{InterfaceName: "lan", Device: "br-lan"},
		},
	})

	resp, err := svc.GetPortList(context.Background())
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if resp == nil || resp.Result == nil || len(resp.Result.Ports) != 2 {
		t.Fatalf("expected two merged ports, got %#v", resp)
	}
	if got := resp.Result.Ports[0].InterfaceNames; len(got) != 2 || got[0] != "existing" || got[1] != "wan" {
		t.Fatalf("expected append-only merge on name match, got %#v", got)
	}
	if got := resp.Result.Ports[1].InterfaceNames; len(got) != 1 || got[0] != "lan" {
		t.Fatalf("expected master-based merge, got %#v", got)
	}
}

func TestServicePropagatesStatusReaderError(t *testing.T) {
	t.Parallel()

	readErr := errors.New("port status failed")
	svc := NewService(&fakeStatusReader{err: readErr}, &fakeMembershipReader{})

	if _, err := svc.GetPortList(context.Background()); !errors.Is(err, readErr) {
		t.Fatalf("expected port status error, got %v", err)
	}
}

func TestServicePropagatesMembershipReaderError(t *testing.T) {
	t.Parallel()

	readErr := errors.New("membership failed")
	svc := NewService(&fakeStatusReader{
		ports: []*models.NetworkPortInfo{{Name: "eth0"}},
	}, &fakeMembershipReader{err: readErr})

	if _, err := svc.GetPortList(context.Background()); !errors.Is(err, readErr) {
		t.Fatalf("expected membership error, got %v", err)
	}
}
