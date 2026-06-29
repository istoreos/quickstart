package service

import (
	"context"
	"errors"
	"testing"
)

func TestGetDiskInfoDelegatesLSBLKOutputToParser(t *testing.T) {
	original := readNasDiskInfoLSBLK
	defer func() {
		readNasDiskInfoLSBLK = original
	}()

	readNasDiskInfoLSBLK = func(ctx context.Context) ([]byte, error) {
		return []byte(`{
			"blockdevices": [
				{
					"name": "sda",
					"path": "/dev/sda",
					"pttype": "dos",
					"type": "disk",
					"size": 2000000000,
					"vendor": "ATA",
					"model": "SSD",
					"children": [
						{"name": "sda1", "path": "/dev/sda1", "size": 1000, "fstype": "ext4", "fsused": "250"}
					]
				}
			]
		}`), nil
	}

	disks, err := getDiskInfo(context.Background())
	if err != nil {
		t.Fatalf("getDiskInfo returned error: %v", err)
	}
	if len(disks) != 1 {
		t.Fatalf("expected one disk, got %d", len(disks))
	}
	if disks[0].Root.Name != "sda" || disks[0].Root.PType != "MBR" {
		t.Fatalf("unexpected root mapping: %#v", disks[0].Root)
	}
	if len(disks[0].Children) != 1 || disks[0].Children[0].Fsused != 250 {
		t.Fatalf("unexpected child mapping: %#v", disks[0].Children)
	}
}

func TestGetDiskInfoPropagatesLSBLKErrors(t *testing.T) {
	original := readNasDiskInfoLSBLK
	defer func() {
		readNasDiskInfoLSBLK = original
	}()

	expectedErr := errors.New("lsblk failed")
	readNasDiskInfoLSBLK = func(ctx context.Context) ([]byte, error) {
		return nil, expectedErr
	}

	if _, err := getDiskInfo(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("expected lsblk error, got %v", err)
	}
}
