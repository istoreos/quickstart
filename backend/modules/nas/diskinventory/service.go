package diskinventory

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/linkease/quick-start/istore-backend/utils"
)

type DiskInfo struct {
	Root     DiskInfoRoot
	Children []*DiskInfoChildren
}

type DiskInfoRoot struct {
	Name        string
	DisplayName string
	Path        string
	PType       string
	TranName    string
	SizeIntStr  string
	SizeStr     string
	Type        string
}

type DiskInfoChildren struct {
	Name          string
	Type          string
	Mountpoint    string
	UUID          string
	ReadOnly      bool
	SizeInt       uint64
	FSType        string
	Fssize        uint64
	Fsused        uint64
	FsusedPercent string
	Path          string
	Label         string
}

type lsblkNasDisk struct {
	Name          string          `json:"name"`
	MountPoint    string          `json:"mountpoint"`
	UUID          string          `json:"uuid"`
	ReadOnly      bool            `json:"ro"`
	Size          uint64          `json:"size"`
	Type          string          `json:"type"`
	Fstype        string          `json:"fstype"`
	Fssize        interface{}     `json:"fssize"`
	Fsused        interface{}     `json:"fsused"`
	FsusedPercent string          `json:"fsuse%"`
	Vendor        string          `json:"vendor"`
	Model         string          `json:"model"`
	Serial        string          `json:"serial"`
	Path          string          `json:"path"`
	Pttype        string          `json:"pttype"`
	Tran          string          `json:"tran"`
	Label         string          `json:"label"`
	Childs        []*lsblkNasDisk `json:"children"`
}

type lsblkNasDisks struct {
	Devices []*lsblkNasDisk `json:"blockdevices"`
}

func ParseLSBLKDisks(raw []byte) ([]*DiskInfo, error) {
	var blk lsblkNasDisks
	if err := json.Unmarshal(raw, &blk); err != nil {
		return nil, err
	}

	disks := make([]*DiskInfo, 0)
	for _, item := range blk.Devices {
		if item.Fstype == "swap" && item.MountPoint == "[SWAP]" {
			continue
		}
		disk := &DiskInfo{}
		if item.Pttype == "dos" {
			item.Pttype = "mbr"
		}
		disk.Root = DiskInfoRoot{
			Name:        item.Name,
			Path:        item.Path,
			PType:       strings.ToUpper(item.Pttype),
			TranName:    item.Tran,
			Type:        item.Type,
			DisplayName: strings.Trim(strings.Trim(item.Vendor, " ")+" "+strings.Trim(item.Model, " ")+" "+strings.Trim(item.Serial, " "), " "),
			SizeStr:     utils.ByteCountBinary(item.Size),
			SizeIntStr:  strconv.FormatUint(item.Size, 10),
		}
		if item.Vendor == "" && item.Model == "" {
			disk.Root.DisplayName = ""
		}

		parts := make([]*DiskInfoChildren, 0)
		for _, child := range item.Childs {
			parts = append(parts, &DiskInfoChildren{
				Name:          child.Name,
				Mountpoint:    child.MountPoint,
				UUID:          child.UUID,
				ReadOnly:      child.ReadOnly,
				SizeInt:       child.Size,
				FSType:        child.Fstype,
				Fssize:        toUint64(child.Fssize),
				Fsused:        toUint64(child.Fsused),
				FsusedPercent: child.FsusedPercent,
				Path:          child.Path,
				Label:         child.Label,
				Type:          item.Type,
			})
		}
		if (item.Fstype != "" || item.MountPoint != "") && item.Pttype == "" {
			disk.Root.PType = "LOOP"
			parts = append(parts, &DiskInfoChildren{
				Name:          item.Name,
				Mountpoint:    item.MountPoint,
				UUID:          item.UUID,
				ReadOnly:      item.ReadOnly,
				SizeInt:       item.Size,
				FSType:        item.Fstype,
				Fssize:        toUint64(item.Fssize),
				Fsused:        toUint64(item.Fsused),
				FsusedPercent: item.FsusedPercent,
				Path:          item.Path,
				Label:         item.Label,
				Type:          item.Type,
			})
		}
		disk.Children = parts
		disks = append(disks, disk)
	}
	return disks, nil
}

func toUint64(src interface{}) uint64 {
	if src == nil {
		return 0
	}
	switch value := src.(type) {
	case int:
		return uint64(value)
	case int8:
		return uint64(value)
	case int16:
		return uint64(value)
	case int32:
		return uint64(value)
	case int64:
		return uint64(value)
	case float32:
		return uint64(value)
	case float64:
		return uint64(value)
	case uint8:
		return uint64(value)
	case uint16:
		return uint64(value)
	case uint32:
		return uint64(value)
	case uint64:
		return value
	case string:
		i, _ := strconv.ParseUint(value, 10, 64)
		return i
	default:
		return 0
	}
}
