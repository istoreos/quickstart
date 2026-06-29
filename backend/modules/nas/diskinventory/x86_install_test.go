package diskinventory

import (
	"context"
	"errors"
	"testing"
)

type fakeX86Store struct {
	rootEnds map[string]int64
	rootErr  error
	bootRoot *X86InstallRoot
	bootErr  error
}

func (store fakeX86Store) ReadRootEnd(ctx context.Context, partitionName string) (int64, error) {
	if store.rootErr != nil {
		return 0, store.rootErr
	}
	return store.rootEnds[partitionName], nil
}

func (store fakeX86Store) ReadFallbackBootRoot(ctx context.Context) (*X86InstallRoot, error) {
	return store.bootRoot, store.bootErr
}

func TestX86InstallSelectsRootAndInstallCandidates(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"blockdevices": [
			{
				"name": "sda",
				"tran": "sata",
				"pttype": "dos",
				"size": 8589934592,
				"vendor": " ATA ",
				"model": " ROOT ",
				"serial": " 001 ",
				"children": [
					{"name": "sda1", "mountpoint": "/boot"},
					{"name": "sda2", "mountpoint": "/"},
					{"name": "sda3", "mountpoint": "/overlay"}
				]
			},
			{
				"name": "sdb",
				"tran": "usb",
				"pttype": "gpt",
				"size": 5368709120,
				"vendor": " USB ",
				"model": " DISK ",
				"serial": " 002 ",
				"children": [
					{"name": "sdb1", "mountpoint": "/mnt/data"},
					{"name": "sdb2"}
				]
			},
			{"name": "loop0", "size": 10737418240},
			{"name": "sr0", "size": 10737418240},
			{"name": "sdc", "size": 1073741824}
		]
	}`)
	svc := NewX86InstallService(fakeX86Store{rootEnds: map[string]int64{"sda3": 123456}})

	result, err := svc.FromLSBLK(context.Background(), raw)
	if err != nil {
		t.Fatalf("FromLSBLK returned error: %v", err)
	}
	if result.Root.Name != "sda" || result.Root.TranName != "sata" || result.Root.End != 123456 || result.Root.PType != "MBR" {
		t.Fatalf("Root = %#v", result.Root)
	}
	if len(result.Devs) != 1 {
		t.Fatalf("Dev count = %d, want 1: %#v", len(result.Devs), result.Devs)
	}
	dev := result.Devs[0]
	if dev.Name != "sdb" || dev.Target != "/dev/sdb" || dev.TranName != "usb" {
		t.Fatalf("Dev identity = %#v", dev)
	}
	if dev.DisplayName != "USB DISK 002" {
		t.Fatalf("DisplayName = %q", dev.DisplayName)
	}
	if dev.SizeStr != "5.0 GiB" {
		t.Fatalf("SizeStr = %q", dev.SizeStr)
	}
	if len(dev.Mountpoints) != 1 || dev.Mountpoints[0] != "/mnt/data" {
		t.Fatalf("Mountpoints = %#v", dev.Mountpoints)
	}
}

func TestX86InstallUsesFallbackBootRoot(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"blockdevices": [
			{"name": "sdb", "tran": "usb", "size": 5368709120}
		]
	}`)
	svc := NewX86InstallService(fakeX86Store{
		bootRoot: &X86InstallRoot{Name: "ventoy", TranName: "usb", End: 4096, PType: "GPT"},
	})

	result, err := svc.FromLSBLK(context.Background(), raw)
	if err != nil {
		t.Fatalf("FromLSBLK returned error: %v", err)
	}
	if result.Root.Name != "ventoy" || result.Root.End != 4096 {
		t.Fatalf("Root = %#v", result.Root)
	}
	if len(result.Devs) != 1 || result.Devs[0].Name != "sdb" {
		t.Fatalf("Devs = %#v", result.Devs)
	}
}

func TestX86InstallReturnsRootNotFound(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"blockdevices": [{"name": "sdb", "size": 5368709120}]}`)
	svc := NewX86InstallService(fakeX86Store{bootErr: errors.New("boot not found")})

	if _, err := svc.FromLSBLK(context.Background(), raw); err == nil || err.Error() != "root not found" {
		t.Fatalf("FromLSBLK error = %v, want root not found", err)
	}
}

func TestX86InstallPropagatesRootEndErrors(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"blockdevices": [
			{
				"name": "sda",
				"pttype": "gpt",
				"size": 8589934592,
				"children": [
					{"name": "sda1", "mountpoint": "/boot"},
					{"name": "sda2"},
					{"name": "sda3"}
				]
			}
		]
	}`)
	expectedErr := errors.New("read root end failed")
	svc := NewX86InstallService(fakeX86Store{rootErr: expectedErr})

	if _, err := svc.FromLSBLK(context.Background(), raw); !errors.Is(err, expectedErr) {
		t.Fatalf("FromLSBLK error = %v, want expectedErr", err)
	}
}
