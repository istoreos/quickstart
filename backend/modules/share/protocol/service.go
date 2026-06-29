package protocol

import (
	"context"
	"errors"
	"strconv"

	"github.com/linkease/quick-start/istore-backend/models"
)

type WebdavRecord struct {
	Port string
}

type SambaRecord struct {
	Workgroup            string
	Description          string
	DisableNetbios       string
	Macos                string
	AllowLegacyProtocols string
}

type Store interface {
	ReadWebdav(ctx context.Context) (WebdavRecord, error)
	UpdateWebdav(ctx context.Context, input models.ShareProtocolWebdavConfig) error
	ReadSamba(ctx context.Context) (SambaRecord, error)
	UpdateSamba(ctx context.Context, input models.ShareProtocolSambaConfig) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) WebdavConfig(ctx context.Context) (*models.ShareProtocolWebdavConfig, error) {
	record, err := svc.store.ReadWebdav(ctx)
	if err != nil {
		return nil, err
	}

	config := &models.ShareProtocolWebdavConfig{}
	if record.Port != "" {
		port, err := strconv.ParseInt(record.Port, 10, 64)
		if err != nil {
			return nil, err
		}
		config.Port = port
	}
	return config, nil
}

func (svc *Service) UpdateWebdav(ctx context.Context, input models.ShareProtocolWebdavConfig) error {
	if input.Port == 0 {
		return errors.New("invalid port")
	}
	return svc.store.UpdateWebdav(ctx, input)
}

func (svc *Service) SambaConfig(ctx context.Context) (*models.ShareProtocolSambaConfig, error) {
	record, err := svc.store.ReadSamba(ctx)
	if err != nil {
		return nil, err
	}

	return &models.ShareProtocolSambaConfig{
		Workgroup:             record.Workgroup,
		Description:           record.Description,
		DisableNetbios:        record.DisableNetbios == "1",
		EnableMacosCompatible: record.Macos == "1",
		AllowLegacy:           record.AllowLegacyProtocols == "1",
	}, nil
}

func (svc *Service) UpdateSamba(ctx context.Context, input models.ShareProtocolSambaConfig) error {
	if input.Workgroup == "" {
		return errors.New("invalid workgroup")
	}
	if input.Description == "" {
		return errors.New("invalid description")
	}
	return svc.store.UpdateSamba(ctx, input)
}
