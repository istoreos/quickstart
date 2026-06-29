package diskinventory

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/linkease/quick-start/istore-backend/utils"
)

type X86Install struct {
	Root X86InstallRoot
	Devs []X86InstallDev
}

type X86InstallRoot struct {
	Name     string
	TranName string
	End      int64
	PType    string
}

type X86InstallDev struct {
	Name        string
	DisplayName string
	Target      string
	TranName    string
	SizeStr     string
	Mountpoints []string
}

type X86InstallStore interface {
	ReadRootEnd(ctx context.Context, partitionName string) (int64, error)
	ReadFallbackBootRoot(ctx context.Context) (*X86InstallRoot, error)
}

type X86InstallService struct {
	store X86InstallStore
}

func NewX86InstallService(store X86InstallStore) *X86InstallService {
	return &X86InstallService{store: store}
}

type x86LSBLKItem struct {
	Name       string          `json:"name"`
	MountPoint string          `json:"mountpoint"`
	Tran       string          `json:"tran"`
	Pttype     string          `json:"pttype"`
	Size       uint64          `json:"size"`
	Vendor     string          `json:"vendor"`
	Model      string          `json:"model"`
	Serial     string          `json:"serial"`
	Childs     []*x86LSBLKItem `json:"children"`
}

type x86LSBLK struct {
	Devices []*x86LSBLKItem `json:"blockdevices"`
}

func (svc *X86InstallService) FromLSBLK(ctx context.Context, raw []byte) (*X86Install, error) {
	var blk x86LSBLK
	if err := json.Unmarshal(raw, &blk); err != nil {
		return nil, err
	}

	x86Install := &X86Install{}
	rootFound := false
	for _, item := range blk.Devices {
		if isX86DiskRoot(item) {
			end, err := svc.store.ReadRootEnd(ctx, item.Childs[2].Name)
			if err != nil {
				return nil, err
			}
			ptype := item.Pttype
			if ptype == "dos" {
				ptype = "mbr"
			}
			x86Install.Root = X86InstallRoot{
				Name:     item.Name,
				TranName: item.Tran,
				End:      end,
				PType:    strings.ToUpper(ptype),
			}
			rootFound = true
			break
		}
	}

	if !rootFound {
		bootRoot, err := svc.store.ReadFallbackBootRoot(ctx)
		if err == nil && bootRoot != nil {
			x86Install.Root = *bootRoot
			rootFound = true
		}
	}

	if !rootFound {
		return nil, errors.New("root not found")
	}

	for _, item := range blk.Devices {
		if item.Name == x86Install.Root.Name ||
			strings.HasPrefix(item.Name, "loop") ||
			strings.HasPrefix(item.Name, "sr") ||
			item.Size <= (2560<<20) {
			continue
		}
		mounts := make([]string, 0)
		for _, child := range item.Childs {
			if child.MountPoint != "" {
				mounts = append(mounts, child.MountPoint)
			}
		}
		x86Install.Devs = append(x86Install.Devs, X86InstallDev{
			Name:        item.Name,
			DisplayName: strings.Trim(strings.Trim(item.Vendor, " ")+" "+strings.Trim(item.Model, " ")+" "+strings.Trim(item.Serial, " "), " "),
			Target:      "/dev/" + item.Name,
			TranName:    item.Tran,
			SizeStr:     utils.ByteCountBinary(item.Size),
			Mountpoints: mounts,
		})
	}
	return x86Install, nil
}

func isX86DiskRoot(item *x86LSBLKItem) bool {
	if len(item.Childs) > 2 {
		for _, child := range item.Childs {
			if child.MountPoint == "/boot" {
				return true
			}
		}
	}
	return false
}
