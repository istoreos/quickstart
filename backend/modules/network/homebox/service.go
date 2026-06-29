package homebox

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type HomeBoxRuntimeChecker interface {
	IsRunning() bool
}

type HomeBoxStarter interface {
	Start(ctx context.Context) error
}

type defaultHomeBoxRuntimeChecker struct{}

func (checker *defaultHomeBoxRuntimeChecker) IsRunning() bool {
	return checkAppIsRunning("homebox")
}

func checkAppIsRunning(name string) bool {
	cmdStr := fmt.Sprintf("pgrep '(^|/)%v[^/]*$'", name)
	isRunning, err := utils.BatchOutput(context.Background(), []string{cmdStr}, 0)
	if err != nil {
		return false
	}
	isRunningStr := strings.Replace(string(isRunning), "\n", "", -1)
	return len(isRunningStr) > 0
}

type defaultHomeBoxStarter struct{}

func (starter *defaultHomeBoxStarter) Start(ctx context.Context) error {
	return utils.BatchRun(ctx, []string{
		fmt.Sprintf("uci set homebox.@homebox[0].enabled=%v", "1"),
		"uci commit homebox",
		"/etc/init.d/homebox restart",
	}, 0)
}

type HomeBoxEnableService struct {
	runtimeChecker HomeBoxRuntimeChecker
	starter        HomeBoxStarter
}

func NewDefaultHomeBoxEnableService() *HomeBoxEnableService {
	return &HomeBoxEnableService{
		runtimeChecker: &defaultHomeBoxRuntimeChecker{},
		starter:        &defaultHomeBoxStarter{},
	}
}

func (svc *HomeBoxEnableService) Enable(ctx context.Context) (*models.NetworkHomeBoxEnableResponse, error) {
	if !svc.runtimeChecker.IsRunning() {
		if err := svc.starter.Start(ctx); err != nil {
			return nil, errors.New("homebox 启动失败")
		}
	}

	resp := &models.NetworkHomeBoxEnableResponse{
		Result: &models.NetworkHomeBoxEnableResponseResult{
			Port: "3300",
		},
	}
	return resp, nil
}
