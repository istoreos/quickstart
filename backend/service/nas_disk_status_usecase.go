package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/nas/diskinventory"
)

type nasDiskStatusInventoryReader struct{}

func (nasDiskStatusInventoryReader) List(ctx context.Context) ([]*diskinventory.DiskInfo, error) {
	return getDiskInfo(ctx)
}

type nasDiskStatusPartitionMarker struct {
	rootPaths        []string
	dockerDevicePath string
}

func (marker nasDiskStatusPartitionMarker) Mark(ctx context.Context, disk *models.NasDiskInfo, partition *models.PartitionInfo) {
	fill_part_status(ctx, disk, partition, marker.rootPaths, marker.dockerDevicePath)
}

type nasDiskStatusRAIDReader struct{}

func (nasDiskStatusRAIDReader) RAIDMember(ctx context.Context, diskName string) string {
	return is_raid_member(diskName)
}

type nasDiskStatusSMARTReader struct{}

func (nasDiskStatusSMARTReader) Config(ctx context.Context) (*models.SmartConfigResponseResult, error) {
	resp, err := SmartGetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (nasDiskStatusSMARTReader) Health(ctx context.Context, diskName string) (string, error) {
	smart, err := get_smart_info(diskName)
	if err != nil {
		return "", err
	}
	return smart.Health, nil
}
