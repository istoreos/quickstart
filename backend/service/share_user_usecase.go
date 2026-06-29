package service

import (
	"context"
	"fmt"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
	shareuser "github.com/linkease/quick-start/istore-backend/modules/share/user"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type shareUserFacade interface {
	List(ctx context.Context) ([]*models.ShareUserInfo, error)
	Create(ctx context.Context, input shareuser.CreateInput) error
	Update(ctx context.Context, input shareuser.UpdateInput) error
	Delete(ctx context.Context, input shareuser.DeleteInput) error
}

var newShareUserService = func() shareUserFacade {
	return shareuser.NewService(defaultShareUserStore{})
}

type defaultShareUserStore struct{}

func (store defaultShareUserStore) ReadUsers(ctx context.Context) ([]*models.ShareUserInfo, error) {
	err := uci.LoadConfig("unishare", true)
	if err != nil {
		return nil, err
	}
	users := make([]*models.ShareUserInfo, 0)
	if sections, ok := uci.GetSections("unishare", "user"); ok {
		for _, section := range sections {
			usr := &models.ShareUserInfo{}
			if value, ok := uci.GetLast("unishare", section, "username"); ok {
				usr.UserName = value
			}
			if value, ok := uci.GetLast("unishare", section, "password"); ok {
				usr.Password = value
			}
			users = append(users, usr)
		}
	}
	return users, nil
}

func (store defaultShareUserStore) CreateUser(ctx context.Context, index int, input shareuser.CreateInput) error {
	target := fmt.Sprintf("@user[%v]", index)
	ucicmdList := []string{
		"add unishare user ",
		fmt.Sprintf("set unishare.%v=%v", target, "user"),
		fmt.Sprintf("set unishare.%v.username=%v", target, input.UserName),
		fmt.Sprintf("set unishare.%v.password=%v", target, input.Password),
		"commit unishare",
	}
	return utils.UCIBatchRun(ctx, ucicmdList, "/etc/init.d/unishare reload", 0)
}

func (store defaultShareUserStore) UpdateUser(ctx context.Context, index int, password string) error {
	target := fmt.Sprintf("@user[%v]", index)
	ucicmdList := []string{
		fmt.Sprintf("set unishare.%v.password=%v", target, password),
		"commit unishare",
	}
	return utils.UCIBatchRun(ctx, ucicmdList, "/etc/init.d/unishare reload", 0)
}

func (store defaultShareUserStore) DeleteUser(ctx context.Context, index int) error {
	target := fmt.Sprintf("@user[%v]", index)
	ucicmdList := []string{
		fmt.Sprintf("del unishare.%v", target),
		"commit unishare",
	}
	return utils.UCIBatchRun(ctx, ucicmdList, "/etc/init.d/unishare reload", 0)
}
