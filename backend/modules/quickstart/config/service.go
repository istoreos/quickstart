package config

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Store interface {
	EnsureConfig(ctx context.Context) error
	Run(ctx context.Context, commands []string) error
	Output(ctx context.Context, commands []string) (string, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (service *Service) Set(ctx context.Context, req models.QuickstartConfigRequest) (*models.SDKNormalResponse, error) {
	if err := service.store.EnsureConfig(ctx); err != nil {
		return nil, err
	}
	if err := service.store.Run(ctx, BuildSetCommands(req)); err != nil {
		return nil, errors.New("设置失败")
	}
	return normalSuccess(), nil
}

func (service *Service) Get(ctx context.Context, req models.QuickstartGetConfigRequest) (*models.QuickstartConfigResponse, error) {
	out, err := service.store.Output(ctx, []string{fmt.Sprintf("uci show quickstart.main.%v", req.Key)})
	if err != nil {
		return nil, errors.New("获取信息失败")
	}
	result, err := ParseGetOutput(req.Key, out)
	if err != nil {
		return nil, err
	}
	return &models.QuickstartConfigResponse{Result: result}, nil
}

func (service *Service) Delete(ctx context.Context, req models.QuickstartDeleteConfigRequest) (*models.SDKNormalResponse, error) {
	if err := service.store.Run(ctx, BuildDeleteCommands(req.Key)); err != nil {
		return nil, errors.New("删除失败")
	}
	return normalSuccess(), nil
}

func (service *Service) GetGlobalFolders(ctx context.Context) (*models.GlobalFoldersResponse, error) {
	if err := service.store.EnsureConfig(ctx); err != nil {
		return nil, err
	}
	out, err := service.store.Output(ctx, []string{"uci show quickstart.main | grep -F 'quickstart.main.' | sed 's/^quickstart\\.main\\.//g'"})
	if err != nil {
		return nil, errors.New("获取信息失败")
	}
	result, err := ParseGlobalFoldersOutput(out)
	if err != nil {
		return nil, err
	}
	return &models.GlobalFoldersResponse{Result: result}, nil
}

func (service *Service) SetGlobalFolders(ctx context.Context, req models.GlobalFolders) (*models.SDKNormalResponse, error) {
	if err := service.store.EnsureConfig(ctx); err != nil {
		return nil, err
	}
	if err := service.store.Run(ctx, BuildGlobalFoldersCommands(req)); err != nil {
		return nil, errors.New("设置失败")
	}
	return normalSuccess(), nil
}

func BuildSetCommands(req models.QuickstartConfigRequest) []string {
	cmds := make([]string, 0, len(req.Values)+1)
	if req.Type == "list" {
		for _, v := range req.Values {
			cmds = append(cmds, fmt.Sprintf("uci add_list quickstart.main.%v=%v", req.Key, v))
		}
	} else {
		for _, v := range req.Values {
			cmds = append(cmds, fmt.Sprintf("uci set quickstart.main.%v=%v", req.Key, v))
		}
	}
	cmds = append(cmds, "uci commit quickstart")
	return cmds
}

func BuildDeleteCommands(key string) []string {
	return []string{
		fmt.Sprintf("uci delete quickstart.main.%v", key),
		"uci commit quickstart",
	}
}

func ParseGetOutput(key string, out string) (*models.QuickstartConfigResponseResult, error) {
	match := regexp.MustCompile(`'(\S+)'`).FindAllStringSubmatch(out, -1)
	if match == nil {
		return nil, errors.New("没有对应的值")
	}
	model := &models.QuickstartConfigResponseResult{Key: key}
	if len(match) > 1 {
		model.Type = "list"
	} else {
		model.Type = "option"
	}
	for _, v := range match {
		model.Values = append(model.Values, v[1])
	}
	return model, nil
}

func BuildGlobalFoldersCommands(req models.GlobalFolders) []string {
	return []string{
		"uci -q batch <<-EOF >/dev/null",
		fmt.Sprintf("set quickstart.main.main_dir=\"%v\"", req.Home),
		fmt.Sprintf("set quickstart.main.conf_dir=\"%v\"", req.Configs),
		fmt.Sprintf("set quickstart.main.pub_dir=\"%v\"", req.Public),
		fmt.Sprintf("set quickstart.main.dl_dir=\"%v\"", req.Downloads),
		fmt.Sprintf("set quickstart.main.tmp_dir=\"%v\"", req.Caches),
		"commit quickstart",
		"EOF",
		"",
	}
}

func ParseGlobalFoldersOutput(out string) (*models.GlobalFolders, error) {
	match := regexp.MustCompile(`(\S+)='(\S+)'`).FindAllStringSubmatch(out, -1)
	if match == nil {
		return nil, errors.New("没有对应的值")
	}
	model := &models.GlobalFolders{}
	for _, v := range match {
		switch v[1] {
		case "main_dir":
			model.Home = v[2]
		case "conf_dir":
			model.Configs = v[2]
		case "pub_dir":
			model.Public = v[2]
		case "dl_dir":
			model.Downloads = v[2]
		case "tmp_dir":
			model.Caches = v[2]
		}
	}
	return model, nil
}

func normalSuccess() *models.SDKNormalResponse {
	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}
}
