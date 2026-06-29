package version

import (
	"context"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Release struct {
	Description string `json:"description"`
	Target      string `json:"target"`
}

type Board struct {
	Kernel  string   `json:"kernel"`
	Model   string   `json:"model"`
	Release *Release `json:"release"`
}

type Store interface {
	ReadBoard(ctx context.Context) (Board, error)
}

type Service struct {
	store             Store
	quickstartVersion string
}

func NewService(store Store, quickstartVersion string) *Service {
	return &Service{
		store:             store,
		quickstartVersion: quickstartVersion,
	}
}

func (svc *Service) Get(ctx context.Context) (*models.SystemVersionResponseResult, error) {
	board, err := svc.store.ReadBoard(ctx)
	if err != nil {
		return nil, err
	}

	firmwareVersion := ""
	if board.Release != nil {
		firmwareVersion = board.Release.Description
	}

	return &models.SystemVersionResponseResult{
		Model:           normalizeModel(board),
		FirmwareVersion: firmwareVersion,
		KernelVersion:   board.Kernel,
		Quickstart:      svc.quickstartVersion,
	}, nil
}

func normalizeModel(board Board) string {
	if board.Release != nil &&
		strings.HasPrefix(board.Release.Target, "x86") &&
		board.Model == "Default string Default string" {
		return "x86 Generic"
	}
	return board.Model
}
