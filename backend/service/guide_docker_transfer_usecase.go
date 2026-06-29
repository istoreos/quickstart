package service

import (
	"context"
	"errors"

	"github.com/linkease/quick-start/istore-backend/models"
	dockertransfer "github.com/linkease/quick-start/istore-backend/modules/guidestorage/dockertransfer"
)

type guideDockerTransferFacade interface {
	GetPartitionList(ctx context.Context) (*models.GuideDockerPartitionListResponse, error)
	Transfer(ctx context.Context, input GuideDockerTransferInput) (*models.GuideDockerTransferResponse, error)
}

var newGuideDockerTransferFacade = func() guideDockerTransferFacade {
	return newGuideDockerTransferService()
}

type GuideDockerTransferInput struct {
	Path         string
	Force        bool
	OverwriteDir bool
}

type GuideDockerTransferService struct {
	reader GuideDockerTransferReader
	writer GuideDockerTransferWriter
}

func newGuideDockerTransferService() *GuideDockerTransferService {
	return &GuideDockerTransferService{
		reader: newDefaultGuideDockerTransferReader(),
		writer: newDefaultGuideDockerTransferWriter(),
	}
}

func (service *GuideDockerTransferService) GetPartitionList(ctx context.Context) (*models.GuideDockerPartitionListResponse, error) {
	candidates, err := service.reader.ReadPartitionCandidates(ctx)
	if err != nil {
		return nil, err
	}
	resp := &models.GuideDockerPartitionListResponse{}
	result := &models.GuideDockerPartitionListResponseResult{PartitionList: make([]string, 0, len(candidates))}
	for _, candidate := range candidates {
		result.PartitionList = append(result.PartitionList, candidate.Path)
	}
	resp.Result = result
	return resp, nil
}

func (service *GuideDockerTransferService) Transfer(ctx context.Context, input GuideDockerTransferInput) (*models.GuideDockerTransferResponse, error) {
	root, err := service.reader.ReadDockerRootPath(ctx)
	if err != nil {
		return nil, err
	}
	if err := service.writer.ValidateTargetPath(ctx, input.Path, root.Path); err != nil {
		return nil, err
	}
	result, err := service.writer.TransferPath(ctx, input.Path, input.Force, input.OverwriteDir, root.Path)
	if err != nil {
		if errors.Is(err, dockertransfer.ErrEmptyTargetDirectory) {
			return &models.GuideDockerTransferResponse{Result: result}, nil
		}
		return nil, err
	}
	if err := service.writer.UpdateDockerRootPath(ctx, input.Path); err != nil {
		return nil, err
	}
	return &models.GuideDockerTransferResponse{}, nil
}
