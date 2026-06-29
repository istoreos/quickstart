package devicelist

import (
	"context"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Reader interface {
	ReadLANInterfaceName(ctx context.Context) (string, error)
	ReadARPForInterface(ctx context.Context, ifname string) (string, error)
}

type Service struct {
	reader Reader
}

func NewService(reader Reader) *Service {
	return &Service{reader: reader}
}

func (svc *Service) List(ctx context.Context) ([]*models.DeviceInfo, error) {
	ifname, err := svc.reader.ReadLANInterfaceName(ctx)
	if err != nil {
		return nil, err
	}
	arpOutput, err := svc.reader.ReadARPForInterface(ctx, ifname)
	if err != nil {
		return nil, err
	}
	return ParseARPDeviceList(arpOutput), nil
}

func ParseARPDeviceList(arpOutput string) []*models.DeviceInfo {
	lines := strings.Split(arpOutput, "\n")
	result := make([]*models.DeviceInfo, 0)
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 6 {
			continue
		}
		if fields[2] != "0x2" {
			continue
		}

		result = append(result, &models.DeviceInfo{
			Ipv4addr: fields[0],
			Name:     "",
			Mac:      strings.ToUpper(strings.TrimSpace(fields[3])),
		})
	}
	return result
}
