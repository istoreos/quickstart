package linkease

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

type ConfigReader interface {
	ReadEnabled(ctx context.Context) (string, error)
	ReadPort(ctx context.Context) (string, error)
}

type ConfigWriter interface {
	Enable(ctx context.Context) error
}

type Service struct {
	reader ConfigReader
	writer ConfigWriter
}

func NewService(reader ConfigReader, writer ConfigWriter) *Service {
	return &Service{
		reader: reader,
		writer: writer,
	}
}

func (svc *Service) Enable(ctx context.Context) (*models.NasLinkeaseEnableResponseResult, error) {
	enabled, err := svc.reader.ReadEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if enabled != "1" {
		if err := svc.writer.Enable(ctx); err != nil {
			return nil, err
		}
	}

	port, err := svc.reader.ReadPort(ctx)
	if err != nil {
		return nil, err
	}

	return &models.NasLinkeaseEnableResponseResult{
		Port: port,
	}, nil
}
