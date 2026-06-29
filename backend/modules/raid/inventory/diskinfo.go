package inventory

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/utils"
)

var partedDeviceKeys = []string{
	"path", "size", "type", "logic_sec", "phy_sec", "p_table", "model", "flags",
}

var partedPartitionKeys = []string{
	"number", "sec_start", "sec_end", "size", "fs", "tag_name", "flags",
}

func BuildDiskInfoFromParted(device string, includeFree bool, partedOutput string) *models.NasDiskInfo {
	disk := &models.NasDiskInfo{}
	lines := strings.Split(partedOutput, "\n")
	diskFields := make(map[string]string)

	for _, line := range lines {
		if strings.HasPrefix(line, "/dev/"+device+":") {
			diskFields = ParsePartedInfo(partedDeviceKeys, line)
			fillDiskInfo(disk, device, diskFields)
			continue
		}
		if !isPartedPartitionLine(line) {
			continue
		}

		partition := buildPartitionInfo(device, includeFree, diskFields, line)
		if partition == nil {
			continue
		}
		disk.Childrens = append(disk.Childrens, partition)
	}

	return disk
}

func fillDiskInfo(disk *models.NasDiskInfo, device string, fields map[string]string) {
	disk.Path = fields["path"]
	disk.Name = device
	if fields["p_table"] == "msdos" {
		disk.PartLabelType = "MBR"
	} else {
		disk.PartLabelType = strings.ToUpper(fields["p_table"])
	}
	disk.VenderModel = fields["model"]
	disk.TranName = fields["type"]

	sizeInt := sectorsToBytes(fields["size"], fields["logic_sec"])
	disk.SizeInt = fmt.Sprintf("%v", sizeInt)
	disk.Size = utils.ByteCountBinary(sizeInt)
}

func buildPartitionInfo(device string, includeFree bool, diskFields map[string]string, line string) *models.PartitionInfo {
	fields := ParsePartedInfo(partedPartitionKeys, line)
	partition := &models.PartitionInfo{
		Filesystem: fields["fs"],
		Name:       "",
		MountPoint: "",
		Usage:      0,
		Used:       "",
	}

	switch {
	case fields["fs"] == "free":
		if !includeFree {
			return nil
		}
		partition.Filesystem = "Free Space"
	case diskFields["p_table"] == "loop":
		partition.Name = device
	default:
		partition.Name = DiskToPart(device, fields["number"])
		partition.Number, _ = strconv.ParseUint(fields["number"], 10, 64)
	}

	sizeInt := sectorsToBytes(fields["size"], diskFields["logic_sec"])
	if fields["size"] != "" {
		partition.SizeInt = fmt.Sprintf("%v", sizeInt)
		partition.Total = utils.ByteCountBinary(sizeInt)
	}
	partition.SecStart, _ = strconv.ParseUint(strings.Trim(fields["sec_start"], "s"), 10, 64)
	partition.SecEnd, _ = strconv.ParseUint(strings.Trim(fields["sec_end"], "s"), 10, 64)

	if strings.Contains(fields["flags"], "raid") {
		partition.IsRaidOn = true
	}
	if fields["fs"] == "" {
		partition.Filesystem = "No FileSystem"
	}

	return partition
}

func sectorsToBytes(size string, sectorSize string) uint64 {
	length, _ := strconv.ParseUint(strings.Trim(size, "s"), 10, 64)
	logicalSectorSize, _ := strconv.ParseUint(sectorSize, 10, 64)
	return length * logicalSectorSize
}

func isPartedPartitionLine(line string) bool {
	if line == "" {
		return false
	}
	first := line[0]
	return first >= '0' && first <= '9' && strings.Contains(line, ":")
}
