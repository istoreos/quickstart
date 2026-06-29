package devicelist

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeReader struct {
	ifname string
	arp    string
	err    error
}

func (reader fakeReader) ReadLANInterfaceName(ctx context.Context) (string, error) {
	if reader.err != nil {
		return "", reader.err
	}
	return reader.ifname, nil
}

func (reader fakeReader) ReadARPForInterface(ctx context.Context, ifname string) (string, error) {
	if reader.err != nil {
		return "", reader.err
	}
	return reader.arp, nil
}

func TestListParsesReachableARPEntries(t *testing.T) {
	t.Parallel()

	arp := `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.10     0x1         0x2         aa:bb:cc:dd:ee:ff     *        br-lan
192.168.1.11     0x1         0x0         11:22:33:44:55:66     *        br-lan
bad line
192.168.1.12     0x1         0x2         77:88:99:aa:bb:cc     *        br-lan
`
	svc := NewService(fakeReader{ifname: "br-lan", arp: arp})

	devices, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	want := []*models.DeviceInfo{
		{Ipv4addr: "192.168.1.10", Name: "", Mac: "AA:BB:CC:DD:EE:FF"},
		{Ipv4addr: "192.168.1.12", Name: "", Mac: "77:88:99:AA:BB:CC"},
	}
	if !reflect.DeepEqual(devices, want) {
		t.Fatalf("devices mismatch\nwant: %#v\n got: %#v", want, devices)
	}
}

func TestListPropagatesReaderErrors(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("read failed")
	svc := NewService(fakeReader{err: expectedErr})

	if _, err := svc.List(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("List error = %v, want expectedErr", err)
	}
}
