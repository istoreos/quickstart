package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Store interface {
	OutputWithErr(ctx context.Context, commands []string) (string, string, error)
	Output(ctx context.Context, commands []string) (string, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (service *Service) StartTest(ctx context.Context, req models.SmartTestRequest) (*models.SmartTestResponse, error) {
	stdout, _, err := service.store.OutputWithErr(ctx, []string{
		fmt.Sprintf("smartctl -t %v %v", req.Type, req.DevicePath),
	})
	model := models.SmartTestResponseResult{}
	if err != nil {
		model.Result = "磁盘测试正在运行\n" + stdout
	} else {
		model.Result = "磁盘检测运行成功"
	}
	return &models.SmartTestResponse{Result: &model}, nil
}

func (service *Service) TestResult(ctx context.Context, req models.SmartTestResultRequest) (*models.SmartTestResultResponse, error) {
	out, err := service.store.Output(ctx, []string{
		fmt.Sprintf("smartctl -l %v %v", req.Type, req.DevicePath),
	})
	if err != nil {
		return nil, errors.New("smart获取测试结果失败")
	}
	model := models.SmartTestResultResponseResult{Result: out}
	return &models.SmartTestResultResponse{Result: &model}, nil
}

func (service *Service) AttributeResult(ctx context.Context, req models.SmartAttributeResultRequest) (*models.SmartAttributeResultResponse, error) {
	out, err := service.store.Output(ctx, []string{
		fmt.Sprintf("smartctl -A %v", req.DevicePath),
	})
	if err != nil {
		return nil, errors.New("smart获取测试结果失败")
	}
	model := models.SmartAttributeResultResponseResult{Result: out}
	return &models.SmartAttributeResultResponse{Result: &model}, nil
}

func (service *Service) ExtendResult(ctx context.Context, req models.SmartExtendResultRequest) (*models.SmartExtendResultResponse, error) {
	out, err := service.store.Output(ctx, []string{
		fmt.Sprintf("smartctl -a %v", req.DevicePath),
	})
	if err != nil {
		return nil, errors.New("smart获取测试结果失败")
	}
	model := models.SmartExtendResultResponseResult{Result: out}
	return &models.SmartExtendResultResponse{Result: &model}, nil
}
