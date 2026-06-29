package status

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Reader interface {
	Read(ctx context.Context) (Snapshot, DNSConfig, error)
}

type Checker interface {
	GetStatus(ip string, gateway string, dns []string) (OnlineStatus, error)
}

type SetupMarker interface {
	MarkSetupFinish(ctx context.Context)
}

type Service struct {
	reader  Reader
	checker Checker
	marker  SetupMarker
}

func NewService(reader Reader, checker Checker, marker SetupMarker) *Service {
	return &Service{
		reader:  reader,
		checker: checker,
		marker:  marker,
	}
}

func (svc *Service) GetNetworkStatus(ctx context.Context, setupFinish bool) (*models.NetworkStatusResponse, error) {
	snapshot, dnsConfig, err := svc.reader.Read(ctx)
	if err != nil {
		return nil, err
	}

	result := buildResult(snapshot, dnsConfig)
	if svc.checker != nil {
		status, err := svc.checker.GetStatus(result.Ipv4addr, result.Gateway, result.DNSList)
		if err != nil {
			return nil, err
		}
		result.NetworkInfo = status.String()
		if setupFinish && svc.marker != nil && status != OnlineDetecting && status != OnlineFailedOffline {
			svc.marker.MarkSetupFinish(ctx)
		}
	}

	return &models.NetworkStatusResponse{Result: result}, nil
}
