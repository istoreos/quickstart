package portlist

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

type StatusReader interface {
	Read(ctx context.Context) ([]*models.NetworkPortInfo, error)
}

type MembershipReader interface {
	Read(ctx context.Context) ([]MembershipSnapshot, error)
}

type Service struct {
	statusReader     StatusReader
	membershipReader MembershipReader
}

func NewService(statusReader StatusReader, membershipReader MembershipReader) *Service {
	return &Service{
		statusReader:     statusReader,
		membershipReader: membershipReader,
	}
}

func (svc *Service) GetPortList(ctx context.Context) (*models.NetworkPortListResponse, error) {
	ports, err := svc.statusReader.Read(ctx)
	if err != nil {
		return nil, err
	}

	memberships, err := svc.membershipReader.Read(ctx)
	if err != nil {
		return nil, err
	}

	return buildResult(mergeMembership(ports, memberships)), nil
}
