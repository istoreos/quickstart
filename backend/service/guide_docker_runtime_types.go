package service

import "github.com/linkease/quick-start/istore-backend/models"

type GuideDockerRuntimeSnapshot struct {
	Installed bool
	Running   bool
	Path      string
	ErrorInfo string
}

func buildGuideDockerRuntimeWarning(disks []*models.NasDiskInfo) string {
	for _, device := range disks {
		if !device.IsDockerRoot {
			continue
		}

		foundDockerPart := false
		for _, part := range device.Childrens {
			if !part.IsDockerRoot {
				continue
			}
			foundDockerPart = true
			if part.IsSystemRoot {
				return "当前docker根目录位于系统根目录，可能会占用大量系统空间，影响系统的正常运行，建议使用docker迁移向导将docker根目录迁移到外置硬盘上"
			}
			if part.Filesystem == "ntfs" {
				return "当前docker根目录位于ntfs分区，会出现很多奇怪的问题，建议迁移到ext4分区"
			}
		}

		if !foundDockerPart {
			return "当前docker根目录位于系统根目录，可能会占用大量系统空间，影响系统的正常运行，建议使用docker迁移向导将docker根目录迁移到外置硬盘上"
		}
		break
	}
	return ""
}
