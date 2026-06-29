package serviceconfig

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

type SambaCreateInput struct {
	ShareName   string
	RootPath    string
	Username    string
	Password    string
	AllowLegacy bool
}

type WebdavCreateInput struct {
	RootPath string
	Username string
	Password string
}

type StatusReader interface {
	ReadSambaShares() []*models.NasServiceSambaInfo
	ReadWebdavPort() (string, bool)
	ReadWebdavInfo() models.NasServiceWebdavInfo
	ReadLinkeaseInfo(ctx context.Context) (enabledByConfig bool, port string, err error)
}

type RuntimeReader interface {
	ReadLANIPv4(ctx context.Context) (string, error)
	HasLinkeaseBinary() bool
}

type ConfigWriter interface {
	PrepareSamba(ctx context.Context) error
	CreateSambaUser(ctx context.Context, username string, password string) error
	WriteSambaShare(ctx context.Context, input SambaCreateInput) error
	WriteWebdavConfig(ctx context.Context, input WebdavCreateInput) error
	RestartWebdav(ctx context.Context) error
}

type SambaTemplateWriter interface {
	EnableRoot() error
}

func BuildSambaURL(ipv4addr string, shareName string) string {
	return "smb://" + ipv4addr + "/" + shareName
}

func BuildWebdavURL(ipv4addr string, port string) string {
	return "http://" + ipv4addr + ":" + port
}
