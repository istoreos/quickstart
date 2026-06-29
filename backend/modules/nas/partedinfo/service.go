package partedinfo

import (
	"context"
	"strconv"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/raid/inventory"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type Store interface {
	RootPaths(ctx context.Context) []string
	DockerDevicePath(ctx context.Context) string
	Parted(ctx context.Context, device string) string
	MountPoint(ctx context.Context, partitionName string) string
	UUID(ctx context.Context, partitionPath string) string
	PartitionUsage(ctx context.Context, partitionName string) (usedKB string, usage string)
	MarkMountedPartition(ctx context.Context, disk *models.NasDiskInfo, partition *models.PartitionInfo, rootPaths []string, dockerDevicePath string)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) Read(ctx context.Context, device string, includeFree bool) (*models.NasDiskInfo, error) {
	rootPaths := svc.store.RootPaths(ctx)
	dockerDevicePath := svc.store.DockerDevicePath(ctx)
	stdout := svc.store.Parted(ctx, device)

	disk := inventory.BuildDiskInfoFromParted(device, includeFree, stdout)
	var diskTotalInt uint64 = 0
	var diskUsedInt uint64 = 0
	partitions := make([]*models.PartitionInfo, 0, len(disk.Childrens))
	for _, partition := range disk.Childrens {
		if partition.Name != "" {
			mountPoint := svc.store.MountPoint(ctx, partition.Name)
			if len(mountPoint) == 0 {
				mountPoint = ""
			}
			partition.MountPoint = mountPoint
			partition.Path = "/dev/" + partition.Name
			uuid := svc.store.UUID(ctx, partition.Path)
			if len(uuid) > 0 {
				partition.UUID = uuid
			}
		}
		if partition.MountPoint != "" && partition.Filesystem != "swap" {
			used, usage := svc.store.PartitionUsage(ctx, partition.Name)
			usedInt, _ := strconv.ParseUint(used, 10, 64)
			usedInt = usedInt * 1024
			usageInt, _ := strconv.ParseUint(usage, 10, 32)
			partition.Used = utils.ByteCountBinary(usedInt)
			partition.Usage = uint32(usageInt)

			sizeInt, _ := strconv.ParseUint(partition.SizeInt, 10, 64)
			diskTotalInt += sizeInt
			diskUsedInt += usedInt
			if diskTotalInt > 0 {
				disk.Usage = uint32(diskUsedInt * 100 / diskTotalInt)
			}
			disk.Total = utils.ByteCountBinary(diskTotalInt)
			disk.Used = utils.ByteCountBinary(diskUsedInt)

			svc.store.MarkMountedPartition(ctx, disk, partition, rootPaths, dockerDevicePath)
		}

		if partition.MountPoint == "/rom" {
			continue
		}

		partitions = append(partitions, partition)
	}
	disk.Childrens = partitions
	return disk, nil
}
