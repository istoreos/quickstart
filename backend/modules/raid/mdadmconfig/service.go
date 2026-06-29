package mdadmconfig

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

type Store interface {
	LoadConfig(ctx context.Context) error
	Arrays(ctx context.Context) []string
	DeleteFirstArray(ctx context.Context) error
	Scan(ctx context.Context) (string, error)
	DiscoverMemberUUIDs(ctx context.Context) (string, error)
	FindFreeMd(min int) int
	AddArray(ctx context.Context, device string, uuid string) error
	Commit(ctx context.Context)
	Enable(ctx context.Context)
	Restart(ctx context.Context)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) Generate(ctx context.Context) error {
	if err := svc.clearExistingArrays(ctx); err != nil {
		return err
	}

	stdout, err := svc.store.Scan(ctx)
	if err != nil {
		return errors.New("mdadm -Ds 命令出错")
	}
	for _, array := range parseScannedArrays(stdout) {
		if err := svc.store.AddArray(ctx, array.device, array.uuid); err != nil {
			return errors.New("添加mdadm配置失败")
		}
	}
	svc.store.Commit(ctx)
	svc.store.Enable(ctx)
	return nil
}

func (svc *Service) AutoFix(ctx context.Context) error {
	stdout, err := svc.store.DiscoverMemberUUIDs(ctx)
	if err != nil {
		return errors.New("获取raid分区信息失败")
	}
	uuids := strings.Split(stdout, "\n")
	if len(uuids) == 0 {
		return errors.New("没有raid分区")
	}

	if err := svc.clearExistingArrays(ctx); err != nil {
		return err
	}

	min := 0
	for _, uuid := range uuids {
		idx := svc.store.FindFreeMd(min)
		if idx == -1 {
			break
		}
		min = idx + 1
		device := "/dev/md" + strconv.Itoa(idx)
		if err := svc.store.AddArray(ctx, device, uuid); err != nil {
			return errors.New("添加mdadm配置失败")
		}
	}
	svc.store.Commit(ctx)
	svc.store.Restart(ctx)
	return nil
}

func (svc *Service) clearExistingArrays(ctx context.Context) error {
	if err := svc.store.LoadConfig(ctx); err != nil {
		return errors.New("获取mdadm配置文件失败")
	}
	for range svc.store.Arrays(ctx) {
		if err := svc.store.DeleteFirstArray(ctx); err != nil {
			return errors.New("删除mdadm配置失败")
		}
	}
	return nil
}

type scannedArray struct {
	device string
	uuid   string
}

func parseScannedArrays(stdout string) []scannedArray {
	lines := strings.Split(stdout, "\n")
	arrays := make([]scannedArray, 0, len(lines))
	for _, line := range lines {
		match := regexp.MustCompile(`^ARRAY ([^\s]+).* UUID=([^\s]+)`).FindStringSubmatch(line)
		if match == nil {
			continue
		}
		arrays = append(arrays, scannedArray{device: match[1], uuid: match[2]})
	}
	return arrays
}
