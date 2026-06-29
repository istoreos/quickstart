package service

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/models"
	smartcommands "github.com/linkease/quick-start/istore-backend/modules/smart/commands"
	smartconfig "github.com/linkease/quick-start/istore-backend/modules/smart/config"
	smartinfo "github.com/linkease/quick-start/istore-backend/modules/smart/info"
	smartinventory "github.com/linkease/quick-start/istore-backend/modules/smart/inventory"
	"github.com/linkease/quick-start/istore-backend/utils"
)

func SmartGetList(ctx context.Context) (*models.SmartListResponse, error) {
	return newSmartInventoryServiceFacade().List(ctx)
}

func SmartGetLog(ctx context.Context) (*models.SmartLogResponse, error) {
	model := models.SmartLogResponseResult{}

	stdout, _, err := utils.BatchOutErr(ctx, []string{"logread -e smartd"}, 0)
	if err != nil {
		return nil, errors.New("获取smart log失败")
	}
	model.Result = stdout
	resp := models.SmartLogResponse{Result: &model}
	return &resp, nil
}

func SmartGetConfig(ctx context.Context) (*models.SmartConfigResponse, error) {
	statPath := "/etc/smartd.conf"
	if canAccessPath(statPath) {
		stat, _ := ioutil.ReadFile(statPath)
		model := smartconfig.ParseSmartConfig(string(stat), CheckAppIsRunning("smartd"))
		resp := models.SmartConfigResponse{Result: model}
		return &resp, nil
	} else {
		return nil, errors.New("smartd.conf文件不存在")
	}
}

// 磁盘热拔插后，自动设置smartd
func SmartReloadDisks() {
	old, _ := SmartGetConfig(context.Background())
	devices, _ := SmartGetList(context.Background())
	deviceConfigs := []*models.SmartConfigDevice{}
	for _, disk := range devices.Result.Disks {
		deviceConfig := models.SmartConfigDevice{DevicePath: disk.Path, TmpDiff: 5, TmpMax: 50}
		for _, v := range old.Result.Devices {
			if v.DevicePath == disk.Path {
				deviceConfig.TmpDiff = v.TmpDiff
				deviceConfig.TmpMax = v.TmpMax
			}
		}
		deviceConfigs = append(deviceConfigs, &deviceConfig)
	}
	req := models.SmartConfigRequest{Global: old.Result.Global, Devices: deviceConfigs, Tasks: old.Result.Tasks}
	smartSetConfig(context.Background(), &req)
}

func smartSetConfig(ctx context.Context, req *models.SmartConfigRequest) {

	//重新生成smard.conf
	confPath := "/etc/smartd.conf"
	for i, line := range smartconfig.RenderSmartdConfig(req) {
		redirect := ">>"
		if i == 0 {
			redirect = ">"
		}
		utils.BatchOutputCmd(ctx, fmt.Sprintf("printf \"%v\n\" %v %v", line, redirect, confPath), 0)
	}
	//更改状态
	if req.Global.Enable {
		utils.BatchOutputCmd(ctx, "/etc/init.d/smartd enable", 0)
		if CheckAppIsRunning("smartd") {
			utils.BatchOutputCmd(ctx, "/etc/init.d/smartd reload", 0)
		} else {
			utils.BatchOutputCmd(ctx, "/etc/init.d/smartd start", 0)
		}
	} else {
		utils.BatchOutputCmd(ctx, "/etc/init.d/smartd disable", 0)
		utils.BatchOutputCmd(ctx, "/etc/init.d/smartd stop", 0)
	}
}

func SmartPostConfig(ctx context.Context, r *http.Request) (*models.SmartConfigResponse, error) {
	req := models.SmartConfigRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return SmartPostConfigTyped(ctx, req)
}

func SmartPostConfigTyped(ctx context.Context, req models.SmartConfigRequest) (*models.SmartConfigResponse, error) {
	smartSetConfig(ctx, &req)

	resp, err := SmartGetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// private
func get_smart_info(device string) (*models.SmartInfo, error) {
	disk, err := get_disk_info(device)
	if err != nil {
		return nil, err
	}
	model := models.SmartInfo{Name: disk.Name, Path: disk.Path, Model: disk.VenderModel, SizeStr: disk.Size}
	cmdlist := []string{
		fmt.Sprintf("smartctl -H -A -i -n standby -f brief /dev/%v", device),
	}
	stdout, _, _ := utils.BatchOutErr(context.Background(), cmdlist, 0)
	return smartinfo.ParseSmartctlInfo(model, stdout), nil
}

func SmartPostTest(ctx context.Context, r *http.Request) (*models.SmartTestResponse, error) {
	req := models.SmartTestRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return SmartPostTestTyped(ctx, req)
}

func SmartPostTestTyped(ctx context.Context, req models.SmartTestRequest) (*models.SmartTestResponse, error) {
	return newSmartCommandServiceFacade().StartTest(ctx, req)
}

func SmartPostTestResult(ctx context.Context, r *http.Request) (*models.SmartTestResultResponse, error) {
	req := models.SmartTestResultRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return SmartPostTestResultTyped(ctx, req)
}

func SmartPostTestResultTyped(ctx context.Context, req models.SmartTestResultRequest) (*models.SmartTestResultResponse, error) {
	return newSmartCommandServiceFacade().TestResult(ctx, req)
}

func SmartPostAttributeResult(ctx context.Context, r *http.Request) (*models.SmartAttributeResultResponse, error) {
	req := models.SmartAttributeResultRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return SmartPostAttributeResultTyped(ctx, req)
}

func SmartPostAttributeResultTyped(ctx context.Context, req models.SmartAttributeResultRequest) (*models.SmartAttributeResultResponse, error) {
	return newSmartCommandServiceFacade().AttributeResult(ctx, req)
}

func SmartPostExtendResult(ctx context.Context, r *http.Request) (*models.SmartExtendResultResponse, error) {
	req := models.SmartExtendResultRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return SmartPostExtendResultTyped(ctx, req)
}

func SmartPostExtendResultTyped(ctx context.Context, req models.SmartExtendResultRequest) (*models.SmartExtendResultResponse, error) {
	return newSmartCommandServiceFacade().ExtendResult(ctx, req)
}

type smartCommandStore struct{}

type smartCommandFacade interface {
	StartTest(ctx context.Context, req models.SmartTestRequest) (*models.SmartTestResponse, error)
	TestResult(ctx context.Context, req models.SmartTestResultRequest) (*models.SmartTestResultResponse, error)
	AttributeResult(ctx context.Context, req models.SmartAttributeResultRequest) (*models.SmartAttributeResultResponse, error)
	ExtendResult(ctx context.Context, req models.SmartExtendResultRequest) (*models.SmartExtendResultResponse, error)
}

var newSmartCommandServiceFacade = func() smartCommandFacade {
	return smartcommands.NewService(smartCommandStore{})
}

func (smartCommandStore) OutputWithErr(ctx context.Context, commands []string) (string, string, error) {
	return utils.BatchOutErr(ctx, commands, 0)
}

func (smartCommandStore) Output(ctx context.Context, commands []string) (string, error) {
	out, err := utils.BatchOutput(ctx, commands, 0)
	return string(out), err
}

type smartInventoryStore struct{}

type smartInventoryFacade interface {
	List(ctx context.Context) (*models.SmartListResponse, error)
}

var newSmartInventoryServiceFacade = func() smartInventoryFacade {
	return smartinventory.NewService(smartInventoryStore{})
}

func (smartInventoryStore) DeviceNames(ctx context.Context) []string {
	files, _ := ioutil.ReadDir("/dev")
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.Name())
	}
	return names
}

func (smartInventoryStore) Scan(ctx context.Context) (string, error) {
	stdout, _, err := utils.BatchOutErr(ctx, []string{"smartctl --scan"}, 0)
	return stdout, err
}

func (smartInventoryStore) Info(ctx context.Context, device string) (*models.SmartInfo, error) {
	return get_smart_info(device)
}
