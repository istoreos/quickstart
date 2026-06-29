package disklifecycle

import (
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
)

type PartitionMountInput struct {
	UUID       string
	Path       string
	MountPoint string
}

type FormatByDevicePathInput struct {
	DevicePath string
}

type InitInput struct {
	Name string
	Path string
}

type InitRestInput struct {
	Name string
	Path string
}

type PartitionSnapshot struct {
	Filesystem string
	MountPoint string
	Name       string
	Path       string
	SecEnd     uint64
	SecStart   uint64
	UUID       string
}

type DiskSnapshot struct {
	Name          string
	PartLabelType string
	Path          string
	Partitions    []PartitionSnapshot
}

func BuildDiskSnapshots(disks []*models.NasDiskInfo) []DiskSnapshot {
	snapshots := make([]DiskSnapshot, 0, len(disks))
	for _, disk := range disks {
		if disk == nil {
			continue
		}
		snapshot := DiskSnapshot{
			Name:          disk.Name,
			PartLabelType: disk.PartLabelType,
			Path:          disk.Path,
			Partitions:    make([]PartitionSnapshot, 0, len(disk.Childrens)),
		}
		for _, partition := range disk.Childrens {
			if partition == nil {
				continue
			}
			snapshot.Partitions = append(snapshot.Partitions, PartitionSnapshot{
				Filesystem: partition.Filesystem,
				MountPoint: partition.MountPoint,
				Name:       partition.Name,
				Path:       partition.Path,
				SecEnd:     partition.SecEnd,
				SecStart:   partition.SecStart,
				UUID:       partition.UUID,
			})
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots
}

func findPartitionByIdentity(disks []DiskSnapshot, uuid string, path string) (*DiskSnapshot, *PartitionSnapshot) {
	for diskIdx := range disks {
		disk := &disks[diskIdx]
		for partitionIdx := range disk.Partitions {
			partition := &disk.Partitions[partitionIdx]
			if partition.UUID == uuid && partition.Path == path {
				return disk, partition
			}
		}
	}
	return nil, nil
}

func isWholeDiskTarget(disks []DiskSnapshot, devicePath string) bool {
	for _, disk := range disks {
		if disk.Path == devicePath {
			return true
		}
	}
	return false
}

func usesDirectDeviceFormat(diskName string) bool {
	return strings.HasPrefix(diskName, "md")
}

func findLastPartition(disk DiskSnapshot) *PartitionSnapshot {
	if len(disk.Partitions) == 0 {
		return nil
	}
	return &disk.Partitions[len(disk.Partitions)-1]
}
