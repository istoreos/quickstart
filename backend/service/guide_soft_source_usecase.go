package service

import (
	"context"
	"errors"

	"github.com/linkease/quick-start/istore-backend/models"
)

type GuideSoftSourceInput struct {
	SoftSourceIdentity string
}

type GuideSoftSourceService struct {
	reader GuideSoftSourceReader
	writer GuideSoftSourceWriter
}

var newGuideSoftSourceServiceFacade = func() *GuideSoftSourceService {
	return newGuideSoftSourceService()
}

func newGuideSoftSourceService() *GuideSoftSourceService {
	return &GuideSoftSourceService{
		reader: newDefaultGuideSoftSourceReader(),
		writer: newDefaultGuideSoftSourceWriter(),
	}
}

var guideSoftSourceList = func(ctx context.Context) (*models.GuideSoftSourceListResponseResult, error) {
	return newGuideSoftSourceServiceFacade().List(ctx)
}

var guideSoftSourceGet = func(ctx context.Context) (*models.GuideSoftSourceResponseResult, error) {
	return newGuideSoftSourceServiceFacade().Get(ctx)
}

var guideSoftSourceSet = func(ctx context.Context, input GuideSoftSourceInput) (*models.GuideSoftSourceResponseResult, error) {
	return newGuideSoftSourceServiceFacade().Set(ctx, input)
}

func (service *GuideSoftSourceService) List(ctx context.Context) (*models.GuideSoftSourceListResponseResult, error) {
	list, err := service.reader.ListSources(ctx)
	if err != nil {
		return nil, err
	}
	return &models.GuideSoftSourceListResponseResult{SoftSourceList: list}, nil
}

func (service *GuideSoftSourceService) Get(ctx context.Context) (*models.GuideSoftSourceResponseResult, error) {
	source, err := service.reader.ReadCurrentSource(ctx)
	if err != nil {
		return nil, err
	}
	return &models.GuideSoftSourceResponseResult{SoftSource: source}, nil
}

func (service *GuideSoftSourceService) Set(ctx context.Context, input GuideSoftSourceInput) (*models.GuideSoftSourceResponseResult, error) {
	source := resolveGuideSoftSourceByIdentity(input.SoftSourceIdentity)
	if len(source.Identity) < 1 {
		return nil, errors.New("没有获取到对应的软件源")
	}
	if err := service.writer.ReplaceSource(ctx, source); err != nil {
		return nil, errors.New("修改软件源失败")
	}
	return service.Get(ctx)
}
