package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

type LanStaticDeviceListService struct {
	LanStatusReader        LanStatusReader
	DhcpTagReader          LanStaticDeviceDhcpTagReader
	StaticAssignmentReader StaticAssignmentListReader
}

func NewLanStaticDeviceListService() *LanStaticDeviceListService {
	dhcpStore := NewDefaultDhcpConfigStore()
	return &LanStaticDeviceListService{
		LanStatusReader:        NewDefaultLanStatusReader(),
		DhcpTagReader:          NewDefaultLanStaticDeviceDhcpTagReader(dhcpStore),
		StaticAssignmentReader: NewDefaultStaticAssignmentListReader(),
	}
}

func (svc *LanStaticDeviceListService) GetListStaticDevices(ctx context.Context) (*models.LANCtrlStaticAssignedResponse, error) {
	lanStatus, err := svc.LanStatusReader.ReadLanStatus(ctx)
	if err != nil {
		return nil, err
	}

	tagList, err := svc.DhcpTagReader.ReadDhcpTags(ctx, lanStatus)
	if err != nil {
		return nil, err
	}

	items, err := svc.StaticAssignmentReader.ReadStaticAssignments(ctx, tagList)
	if err != nil {
		return nil, err
	}

	return &models.LANCtrlStaticAssignedResponse{
		Result: items,
	}, nil
}
