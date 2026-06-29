package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
	appcore "github.com/linkease/quick-start/istore-backend/modules/app/core"
	appmetadata "github.com/linkease/quick-start/istore-backend/modules/app/metadata"
	quickstartconfig "github.com/linkease/quick-start/istore-backend/modules/quickstart/config"
	"github.com/linkease/quick-start/istore-backend/utils"
)

func AppCheck(ctx context.Context, r *http.Request) (*models.AppCheckResponse, error) {
	req := models.AppCheckRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return AppCheckValue(ctx, req)
}

func AppCheckValue(ctx context.Context, req models.AppCheckRequest) (*models.AppCheckResponse, error) {
	return newAppServiceFacade().Check(ctx, req)
}

func AppInstall(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.AppInstallRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return AppInstallValue(ctx, req)
}

func AppInstallValue(ctx context.Context, req models.AppInstallRequest) (*models.SDKNormalResponse, error) {
	return newAppServiceFacade().Install(ctx, req)
}

// 检查是否已安装插件
// opkg find name
func CheckAppIsInstalled(name string) (bool, error) {
	//忽略其他软件配置的错误输出
	return canAccessPath(fmt.Sprintf("/usr/lib/opkg/info/%v.control", name)), nil
	// cmd := exec.Command("opkg", "-V0", "find", name)
	// var out bytes.Buffer
	// var errbuf bytes.Buffer
	// cmd.Stdout = &out
	// cmd.Stderr = &errbuf
	// err := cmd.Run()
	// if err != nil {
	// 	return canAccessPath(fmt.Sprintf("/usr/lib/opkg/info/%v.control", name)), nil
	// 	// return false, errors.New("opkg find " + name + " err")
	// }
	// if errbuf.String() != "" {
	// 	return false, errors.New(errbuf.String())
	// }
	// if out.String() != "" {
	// 	return true, nil
	// }
	// return false, errors.New("not found " + name)
}

// 安装插件
// opkg install name
func InstallApp(name string) (string, error) {
	go func() {
		cmd := exec.Command("is-opkg", "install", name)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Run()
	}()

	return "installing", nil
}

func CheckAppIsRunning(name string) bool {
	cmdStr := fmt.Sprintf("pgrep '(^|/)%v[^/]*$'", name)
	isRunning, err := utils.BatchOutput(context.Background(), []string{cmdStr}, 0)
	if err != nil {
		return false
	}
	isRunningStr := strings.Replace(string(isRunning), "\n", "", -1)
	return len(isRunningStr) > 0
}

func QuickstartSetConfigValue(ctx context.Context, req models.QuickstartConfigRequest) (*models.SDKNormalResponse, error) {
	return newQuickstartConfigServiceFacade().Set(ctx, req)
}

func QuickstartGetConfigValue(ctx context.Context, req models.QuickstartGetConfigRequest) (*models.QuickstartConfigResponse, error) {
	return newQuickstartConfigServiceFacade().Get(ctx, req)
}

func QuickstartDeleteConfigValue(ctx context.Context, req models.QuickstartDeleteConfigRequest) (*models.SDKNormalResponse, error) {
	return newQuickstartConfigServiceFacade().Delete(ctx, req)
}

func QuickstartSetConfig(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.QuickstartConfigRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return QuickstartSetConfigValue(ctx, req)
}

func QuickstartGetConfig(ctx context.Context, r *http.Request) (*models.QuickstartConfigResponse, error) {
	req := models.QuickstartGetConfigRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return QuickstartGetConfigValue(ctx, req)
}

func QuickstartDeleteConfig(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.QuickstartDeleteConfigRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return QuickstartDeleteConfigValue(ctx, req)
}

func checkQuickstartConfig() error {
	configPath := "/etc/config/quickstart"
	if !canAccessPath(configPath) {
		return errors.New("无法访问/etc/config/quickstart")
	}
	return nil
}

func GlobalFoldersGetConfig(ctx context.Context) (*models.GlobalFoldersResponse, error) {
	return newQuickstartConfigServiceFacade().GetGlobalFolders(ctx)
}

func GlobalFoldersPostConfig(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.GlobalFolders{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return newQuickstartConfigServiceFacade().SetGlobalFolders(ctx, req)
}

type quickstartConfigStore struct{}

type quickstartConfigFacade interface {
	Set(ctx context.Context, req models.QuickstartConfigRequest) (*models.SDKNormalResponse, error)
	Get(ctx context.Context, req models.QuickstartGetConfigRequest) (*models.QuickstartConfigResponse, error)
	Delete(ctx context.Context, req models.QuickstartDeleteConfigRequest) (*models.SDKNormalResponse, error)
	GetGlobalFolders(ctx context.Context) (*models.GlobalFoldersResponse, error)
	SetGlobalFolders(ctx context.Context, req models.GlobalFolders) (*models.SDKNormalResponse, error)
}

var newQuickstartConfigServiceFacade = func() quickstartConfigFacade {
	return quickstartconfig.NewService(quickstartConfigStore{})
}

func (quickstartConfigStore) EnsureConfig(ctx context.Context) error {
	return checkQuickstartConfig()
}

func (quickstartConfigStore) Run(ctx context.Context, commands []string) error {
	return utils.BatchRun(ctx, commands, 0)
}

func (quickstartConfigStore) Output(ctx context.Context, commands []string) (string, error) {
	out, _, err := utils.BatchOutErr(ctx, commands, 0)
	return out, err
}

func AppInstalledList(ctx context.Context, r *http.Request) (models.AppInstalledListResponse, error) {
	return AppInstalledListValue(ctx)
}

func AppInstalledListValue(ctx context.Context) (models.AppInstalledListResponse, error) {
	return newAppServiceFacade().InstalledList(ctx)
}

type appStore struct{}

type appServiceFacade interface {
	Check(ctx context.Context, req models.AppCheckRequest) (*models.AppCheckResponse, error)
	Install(ctx context.Context, req models.AppInstallRequest) (*models.SDKNormalResponse, error)
	InstalledList(ctx context.Context) (models.AppInstalledListResponse, error)
}

var newAppServiceFacade = func() appServiceFacade {
	return appcore.NewService(appStore{})
}

func (appStore) IsInstalled(ctx context.Context, name string) (bool, error) {
	return CheckAppIsInstalled(name)
}

func (appStore) IsRunning(ctx context.Context, name string) bool {
	return CheckAppIsRunning(name)
}

func (appStore) Install(ctx context.Context, name string) (string, error) {
	return InstallApp(name)
}

func (appStore) InstalledList(ctx context.Context) ([]*models.AppInstalled, error) {
	return readApplistFromPath("/usr/lib/opkg/meta")
}

func readApplistFromPath(p string) ([]*models.AppInstalled, error) {
	return appmetadata.NewService(appMetadataStore{}).List(p)
}

type appMetadataStore struct{}

func (appMetadataStore) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

func (appMetadataStore) Stat(path string) (appmetadata.FileInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return appmetadata.FileInfo{}, err
	}
	return appmetadata.FileInfo{ModTimeUnix: fi.ModTime().Unix()}, nil
}

func (appMetadataStore) ReadFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}
