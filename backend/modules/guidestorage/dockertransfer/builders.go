package dockertransfer

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/linkease/quick-start/istore-backend/models"
)

var ErrEmptyTargetDirectory = errors.New("目标路径不为空")

type PartitionCandidate struct {
	MountPoint string
	Filesystem string
	SizeBytes  uint64
	Path       string
}

func BuildUpdateCommands(path string) []string {
	return []string{
		fmt.Sprintf("uci set dockerd.globals.data_root='%v'", path),
		"uci commit dockerd",
		"/etc/init.d/dockerd restart",
	}
}

func BuildPartitionCandidates(disks []*models.NasDiskInfo) []*PartitionCandidate {
	candidates := make([]*PartitionCandidate, 0)
	for _, device := range disks {
		for _, part := range device.Childrens {
			if part.IsSystemRoot || part.IsReadOnly || part.MountPoint == "" {
				continue
			}
			switch part.Filesystem {
			case "squashfs", "ntfs", "vfat", "exfat", "swap":
				continue
			}
			sizeBytes, _ := strconv.ParseUint(part.SizeInt, 10, 64)
			if sizeBytes <= 8*1024*1024*1024 {
				continue
			}
			candidates = append(candidates, &PartitionCandidate{
				MountPoint: part.MountPoint,
				Filesystem: part.Filesystem,
				SizeBytes:  sizeBytes,
				Path:       part.MountPoint + "/docker",
			})
		}
	}
	return candidates
}

func BuildEmptyTargetDirectoryWarning(path string) (*models.GuideDockerTransferResponseResult, error) {
	return &models.GuideDockerTransferResponseResult{
		Path:             path,
		EmptyPathWarning: true,
	}, ErrEmptyTargetDirectory
}
