package service

import (
	"github.com/linkease/quick-start/istore-backend/models"
	disklifecycle "github.com/linkease/quick-start/istore-backend/modules/nas/disklifecycle"
)

type NasDiskPartitionMountInput = disklifecycle.PartitionMountInput
type NasDiskFormatByDevicePathInput = disklifecycle.FormatByDevicePathInput
type NasDiskInitInput = disklifecycle.InitInput
type NasDiskInitRestInput = disklifecycle.InitRestInput

type NasDiskLifecyclePartitionSnapshot = disklifecycle.PartitionSnapshot
type NasDiskLifecycleDiskSnapshot = disklifecycle.DiskSnapshot

func buildNasDiskLifecycleDiskSnapshots(disks []*models.NasDiskInfo) []NasDiskLifecycleDiskSnapshot {
	return disklifecycle.BuildDiskSnapshots(disks)
}
