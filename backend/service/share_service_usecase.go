package service

import (
	"context"
	"fmt"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
	shareservice "github.com/istoreos/quickstart/backend/modules/share/service"
	"github.com/istoreos/quickstart/backend/utils"
)

type shareServiceFacade interface {
	List(ctx context.Context) ([]*models.ShareServiceInfo, error)
	Create(ctx context.Context, input shareservice.CreateInput) error
	Ensure(ctx context.Context, input shareservice.CreateInput) error
	Update(ctx context.Context, input shareservice.UpdateInput) error
	Delete(ctx context.Context, input shareservice.DeleteInput) error
}

var newShareService = func() shareServiceFacade {
	return shareservice.NewService(defaultShareServiceStore{})
}

type defaultShareServiceStore struct{}

func (store defaultShareServiceStore) ReadConfig(ctx context.Context) ([]*shareservice.ShareRecord, []*models.ShareUserInfo, error) {
	if err := uci.LoadConfig("unishare", true); err != nil {
		return nil, nil, err
	}

	shares := make([]*shareservice.ShareRecord, 0)
	if sections, ok := uci.GetSections("unishare", "share"); ok {
		for _, section := range sections {
			share := &shareservice.ShareRecord{}
			if value, ok := uci.GetLast("unishare", section, "path"); ok {
				share.Path = value
			}
			if value, ok := uci.GetLast("unishare", section, "name"); ok {
				share.Name = value
			}
			if value, ok := uci.Get("unishare", section, "ro"); ok {
				share.RO = value
			}
			if value, ok := uci.Get("unishare", section, "rw"); ok {
				share.RW = value
			}
			if value, ok := uci.Get("unishare", section, "proto"); ok {
				share.Proto = value
			}
			shares = append(shares, share)
		}
	}

	users := make([]*models.ShareUserInfo, 0)
	if sections, ok := uci.GetSections("unishare", "user"); ok {
		for _, section := range sections {
			if value, ok := uci.GetLast("unishare", section, "username"); ok {
				users = append(users, &models.ShareUserInfo{UserName: value})
			}
		}
	}

	return shares, users, nil
}

func (store defaultShareServiceStore) CreateShare(ctx context.Context, index int, input shareservice.CreateInput) error {
	target := fmt.Sprintf("@share[%v]", index)
	ucicmdList := []string{
		"add unishare share ",
		"set unishare.@global[0].enabled=1",
		fmt.Sprintf("set unishare.%v=%v", target, "share"),
		fmt.Sprintf("set unishare.%v.path=%v", target, input.Path),
		fmt.Sprintf("set unishare.%v.name=%v", target, input.Name),
	}
	ucicmdList = append(ucicmdList, shareServiceListCommands(target, input)...)
	ucicmdList = append(ucicmdList, "commit unishare")

	return utils.UCIBatchRun(ctx, ucicmdList, "/etc/init.d/unishare reload", 0)
}

func (store defaultShareServiceStore) UpdateShare(ctx context.Context, index int, input shareservice.UpdateInput) error {
	target := fmt.Sprintf("@share[%v]", index)
	ucicmdList := []string{
		fmt.Sprintf("set unishare.%v=%v", target, "share"),
		fmt.Sprintf("set unishare.%v.path=%v", target, input.Path),
		fmt.Sprintf("set unishare.%v.name=%v", target, input.Name),
		fmt.Sprintf("del unishare.%v.ro", target),
		fmt.Sprintf("del unishare.%v.rw", target),
		fmt.Sprintf("del unishare.%v.proto", target),
	}
	ucicmdList = append(ucicmdList, shareServiceListCommands(target, input)...)
	ucicmdList = append(ucicmdList, "commit unishare")

	return utils.UCIBatchRun(ctx, ucicmdList, "/etc/init.d/unishare reload", 0)
}

func (store defaultShareServiceStore) DeleteShare(ctx context.Context, index int) error {
	target := fmt.Sprintf("@share[%v]", index)
	ucicmdList := []string{
		fmt.Sprintf("del unishare.%v", target),
		"commit unishare",
	}

	return utils.UCIBatchRun(ctx, ucicmdList, "/etc/init.d/unishare reload", 0)
}

func shareServiceListCommands(target string, input shareservice.CreateInput) []string {
	ro := make([]string, 0)
	rw := make([]string, 0)
	for _, user := range input.Users {
		if user.Rw {
			rw = append(rw, user.UserName)
		}
		if user.Ro {
			ro = append(ro, user.UserName)
		}
	}

	proto := make([]string, 0)
	if input.Samba {
		proto = append(proto, "samba")
	}
	if input.Webdav {
		proto = append(proto, "webdav")
	}

	commands := make([]string, 0, len(ro)+len(rw)+len(proto))
	for _, value := range ro {
		commands = append(commands, fmt.Sprintf("add_list unishare.%v.ro=%v", target, value))
	}
	for _, value := range rw {
		commands = append(commands, fmt.Sprintf("add_list unishare.%v.rw=%v", target, value))
	}
	for _, value := range proto {
		commands = append(commands, fmt.Sprintf("add_list unishare.%v.proto=%v", target, value))
	}
	return commands
}

func ShareServiceEnsureTyped(ctx context.Context, req models.ShareServiceCreateRequest) (*models.SDKNormalResponse, error) {
	if err := newShareService().Ensure(ctx, shareservice.CreateInput{Name: req.Name, Path: req.Path, Samba: req.Samba, Webdav: req.Webdav, Users: req.Users}); err != nil {
		return nil, err
	}
	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}
