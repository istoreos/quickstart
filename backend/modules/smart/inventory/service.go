package inventory

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Store interface {
	DeviceNames(ctx context.Context) []string
	Scan(ctx context.Context) (string, error)
	Info(ctx context.Context, device string) (*models.SmartInfo, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (service *Service) List(ctx context.Context) (*models.SmartListResponse, error) {
	model := models.SmartListResponseResult{}
	diskNames := FilterCandidateDeviceNames(service.store.DeviceNames(ctx))

	// smartctl reports some NVMe device names as nvme0 while /dev uses nvme0n1.
	stdout, err := service.store.Scan(ctx)
	if err != nil {
		return nil, errors.New("获取smart设备列表失败")
	}
	for _, name := range diskNames {
		if strings.Contains(stdout, name) || strings.Contains(name, "nvme") {
			smart, err := service.store.Info(ctx, name)
			if err != nil {
				return nil, err
			}
			model.Disks = append(model.Disks, smart)
		}
	}
	return &models.SmartListResponse{Result: &model}, nil
}

func FilterCandidateDeviceNames(deviceNames []string) []string {
	candidates := make([]string, 0, len(deviceNames))
	for _, dev := range deviceNames {
		if isCandidateDeviceName(dev) {
			candidates = append(candidates, dev)
		}
	}
	return candidates
}

func isCandidateDeviceName(dev string) bool {
	return regexp.MustCompile(`^sd[a-z]$`).MatchString(dev) ||
		regexp.MustCompile(`^mmcblk\d+$`).MatchString(dev) ||
		regexp.MustCompile(`^sata[a-z]$`).MatchString(dev) ||
		regexp.MustCompile(`^nvme\d+n\d+$`).MatchString(dev)
}
