package service

import (
	"context"
	"errors"
	"testing"

	simplejson "github.com/bitly/go-simplejson"
	"github.com/linkease/quick-start/istore-backend/models"
)

func TestDefaultGuideStatusReadsReaderReadDdnstoConfig(t *testing.T) {
	prevBatch := readGuideStatusReadsBatchOutErr
	defer func() { readGuideStatusReadsBatchOutErr = prevBatch }()
	readGuideStatusReadsBatchOutErr = func(ctx context.Context, cmds []string, timeout int) (string, string, error) {
		if len(cmds) != 1 || cmds[0] != "/usr/sbin/ddnsto -w | awk '{print $2}'" {
			t.Fatalf("unexpected ddnsto config commands: %#v", cmds)
		}
		return "device-123", "", nil
	}

	reader := newDefaultGuideStatusReadsReader()
	snapshot, err := reader.ReadDdnstoConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected ddnsto config error: %v", err)
	}
	if snapshot.DeviceID != "device-123" {
		t.Fatalf("unexpected device id: %#v", snapshot)
	}
}

func TestDefaultGuideStatusReadsReaderReadDDNSStatus(t *testing.T) {
	prevUbus := readGuideStatusReadsUbusCall
	prevInstalled := readGuideStatusReadsCheckAppIsInstalled
	prevBatch := readGuideStatusReadsBatchOutErr
	prevUciGetLast := readGuideStatusReadsUciGetLast
	defer func() {
		readGuideStatusReadsUbusCall = prevUbus
		readGuideStatusReadsCheckAppIsInstalled = prevInstalled
		readGuideStatusReadsBatchOutErr = prevBatch
		readGuideStatusReadsUciGetLast = prevUciGetLast
	}()

	readGuideStatusReadsUbusCall = func(ctx context.Context, arg string) (*simplejson.Json, error) {
		if arg != "luci.ddns get_services_status" {
			t.Fatalf("unexpected ubus arg: %s", arg)
		}
		json := simplejson.New()
		json.SetPath([]string{"myddns_ipv4", "next_update"}, "Stopped")
		json.SetPath([]string{"myddns_ipv6", "next_update"}, "Running")
		return json, nil
	}
	readGuideStatusReadsCheckAppIsInstalled = func(name string) (bool, error) {
		if name != "ddnsto" {
			t.Fatalf("unexpected app name: %s", name)
		}
		return true, nil
	}
	readGuideStatusReadsBatchOutErr = func(ctx context.Context, cmds []string, timeout int) (string, string, error) {
		if len(cmds) == 1 && cmds[0] == "uci get ddnsto.@ddnsto[0].address" {
			return "https://demo.example.com:443", "", nil
		}
		t.Fatalf("unexpected ddns status cmds: %#v", cmds)
		return "", "", nil
	}

	readGuideStatusReadsUciGetLast = func(config string, section string, option string) (string, bool) {
		if config == "ddns" && section == "myddns_ipv6" && option == "lookup_host" {
			return "ipv6.example.com", true
		}
		return "", false
	}

	reader := newDefaultGuideStatusReadsReader()
	snapshot, err := reader.ReadDDNSStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected ddns status error: %v", err)
	}
	if snapshot.IPV4Domain != "Stopped" || snapshot.IPV6Domain != "ipv6.example.com" || snapshot.DdnstoDomain != "https://demo.example.com:443" {
		t.Fatalf("unexpected ddns snapshot: %#v", snapshot)
	}
}

func TestDefaultGuideStatusReadsReaderReadDDNSStatusBuildsDdnstoFallbackAddress(t *testing.T) {
	prevUbus := readGuideStatusReadsUbusCall
	prevInstalled := readGuideStatusReadsCheckAppIsInstalled
	prevBatch := readGuideStatusReadsBatchOutErr
	defer func() {
		readGuideStatusReadsUbusCall = prevUbus
		readGuideStatusReadsCheckAppIsInstalled = prevInstalled
		readGuideStatusReadsBatchOutErr = prevBatch
	}()

	readGuideStatusReadsUbusCall = func(ctx context.Context, arg string) (*simplejson.Json, error) {
		return nil, errors.New("ignore ddns ubus")
	}
	readGuideStatusReadsCheckAppIsInstalled = func(name string) (bool, error) {
		return true, nil
	}
	call := 0
	readGuideStatusReadsBatchOutErr = func(ctx context.Context, cmds []string, timeout int) (string, string, error) {
		call++
		switch call {
		case 1:
			if cmds[0] != "uci get ddnsto.@ddnsto[0].address" {
				t.Fatalf("unexpected first ddnsto cmds: %#v", cmds)
			}
			return "", "", nil
		case 2:
			if cmds[0] != "/usr/sbin/ddnsto -w | awk '{print $2}'" {
				t.Fatalf("unexpected second ddnsto cmds: %#v", cmds)
			}
			return "device-abc", "", nil
		case 3:
			if len(cmds) != 2 {
				t.Fatalf("unexpected fallback update cmds: %#v", cmds)
			}
			return "", "", nil
		default:
			t.Fatalf("unexpected batch call count: %d", call)
			return "", "", nil
		}
	}

	reader := newDefaultGuideStatusReadsReader()
	snapshot, err := reader.ReadDDNSStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected ddns fallback error: %v", err)
	}
	if snapshot.DdnstoDomain == "" {
		t.Fatalf("expected generated ddnsto domain, got %#v", snapshot)
	}
}

func TestDefaultGuideStatusReadsReaderReadDDNSStatusPropagatesInstallCheckError(t *testing.T) {
	prevInstalled := readGuideStatusReadsCheckAppIsInstalled
	defer func() { readGuideStatusReadsCheckAppIsInstalled = prevInstalled }()
	readGuideStatusReadsCheckAppIsInstalled = func(name string) (bool, error) {
		return false, errors.New("install check failed")
	}

	reader := newDefaultGuideStatusReadsReader()
	if _, err := reader.ReadDDNSStatus(context.Background()); err == nil || err.Error() != "install check failed" {
		t.Fatalf("unexpected install-check error: %v", err)
	}
}

func TestDefaultGuideStatusReadsReaderReadDownloadPartitions(t *testing.T) {
	prevDisks := readGuideStatusReadsAllDisks
	defer func() { readGuideStatusReadsAllDisks = prevDisks }()
	readGuideStatusReadsAllDisks = func(ctx context.Context) ([]*models.NasDiskInfo, error) {
		return []*models.NasDiskInfo{
			{
				Name: "sda",
				Childrens: []*models.PartitionInfo{
					{MountPoint: "/mnt/data1", Filesystem: "ext4"},
					{MountPoint: "/mnt/root", Filesystem: "ext4", IsSystemRoot: true},
					{MountPoint: "/mnt/ro", Filesystem: "ext4", IsReadOnly: true},
					{MountPoint: "/mnt/sq", Filesystem: "squashfs"},
					{MountPoint: "/mnt/swap", Filesystem: "swap"},
					{MountPoint: "", Filesystem: "ext4"},
				},
			},
		}, nil
	}

	reader := newDefaultGuideStatusReadsReader()
	partitions, err := reader.ReadDownloadPartitions(context.Background())
	if err != nil {
		t.Fatalf("unexpected partition read error: %v", err)
	}
	if len(partitions) != 1 || partitions[0] != "/mnt/data1/download" {
		t.Fatalf("unexpected partition list: %#v", partitions)
	}
}
