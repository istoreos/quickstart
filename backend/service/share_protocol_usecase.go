package service

import (
	"context"
	"fmt"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
	shareprotocol "github.com/linkease/quick-start/istore-backend/modules/share/protocol"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type shareProtocolFacade interface {
	WebdavConfig(ctx context.Context) (*models.ShareProtocolWebdavConfig, error)
	UpdateWebdav(ctx context.Context, input models.ShareProtocolWebdavConfig) error
	SambaConfig(ctx context.Context) (*models.ShareProtocolSambaConfig, error)
	UpdateSamba(ctx context.Context, input models.ShareProtocolSambaConfig) error
}

var newShareProtocol = func() shareProtocolFacade {
	return shareprotocol.NewService(defaultShareProtocolStore{})
}

type defaultShareProtocolStore struct{}

func (store defaultShareProtocolStore) ReadWebdav(ctx context.Context) (shareprotocol.WebdavRecord, error) {
	if err := uci.LoadConfig("unishare", true); err != nil {
		return shareprotocol.WebdavRecord{}, err
	}

	record := shareprotocol.WebdavRecord{}
	if value, ok := uci.GetLast("unishare", "@global[0]", "webdav_port"); ok {
		record.Port = value
	}
	return record, nil
}

func (store defaultShareProtocolStore) UpdateWebdav(ctx context.Context, input models.ShareProtocolWebdavConfig) error {
	ucicmdList := []string{
		fmt.Sprintf("set unishare.@global[0].webdav_port=%v", input.Port),
		"commit unishare",
	}
	return utils.UCIBatchRun(ctx, ucicmdList, "/etc/init.d/unishare reload", 0)
}

func (store defaultShareProtocolStore) ReadSamba(ctx context.Context) (shareprotocol.SambaRecord, error) {
	if err := uci.LoadConfig("samba4", true); err != nil {
		return shareprotocol.SambaRecord{}, err
	}

	record := shareprotocol.SambaRecord{}
	if value, ok := uci.GetLast("samba4", "@samba[0]", "workgroup"); ok {
		record.Workgroup = value
	}
	if value, ok := uci.GetLast("samba4", "@samba[0]", "description"); ok {
		record.Description = value
	}
	if value, ok := uci.GetLast("samba4", "@samba[0]", "disable_netbios"); ok {
		record.DisableNetbios = value
	}
	if value, ok := uci.GetLast("samba4", "@samba[0]", "macos"); ok {
		record.Macos = value
	}
	if value, ok := uci.GetLast("samba4", "@samba[0]", "allow_legacy_protocols"); ok {
		record.AllowLegacyProtocols = value
	}
	return record, nil
}

func (store defaultShareProtocolStore) UpdateSamba(ctx context.Context, input models.ShareProtocolSambaConfig) error {
	ucicmdList := []string{
		fmt.Sprintf("set samba4.@samba[0].workgroup=%v", input.Workgroup),
		fmt.Sprintf("set samba4.@samba[0].description=%v", input.Description),
	}

	if input.DisableNetbios {
		ucicmdList = append(ucicmdList, fmt.Sprintf("set samba4.@samba[0].disable_netbios=%v", 1))
	} else {
		ucicmdList = append(ucicmdList, "del samba4.@samba[0].disable_netbios")
	}

	if input.EnableMacosCompatible {
		ucicmdList = append(ucicmdList, fmt.Sprintf("set samba4.@samba[0].macos=%v", 1))
	} else {
		ucicmdList = append(ucicmdList, "del samba4.@samba[0].macos")
	}

	if input.AllowLegacy {
		ucicmdList = append(ucicmdList, fmt.Sprintf("set samba4.@samba[0].allow_legacy_protocols=%v", 1))
	} else {
		ucicmdList = append(ucicmdList, "del samba4.@samba[0].allow_legacy_protocols")
	}
	ucicmdList = append(ucicmdList, "commit samba4")

	return utils.UCIBatchRun(ctx, ucicmdList, "/etc/init.d/samba4 reload", 0)
}
