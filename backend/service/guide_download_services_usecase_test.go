package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeGuideDownloadServicesReader struct {
	aria2           *GuideDownloadAria2Snapshot
	aria2Err        error
	qbittorrent     *GuideDownloadQbittorrentSnapshot
	qbittorrentErr  error
	transmission    *GuideDownloadTransmissionSnapshot
	transmissionErr error
}

func (reader *fakeGuideDownloadServicesReader) ReadAria2Status(ctx context.Context) (*GuideDownloadAria2Snapshot, error) {
	return reader.aria2, reader.aria2Err
}

func (reader *fakeGuideDownloadServicesReader) ReadQbittorrentStatus(ctx context.Context) (*GuideDownloadQbittorrentSnapshot, error) {
	return reader.qbittorrent, reader.qbittorrentErr
}

func (reader *fakeGuideDownloadServicesReader) ReadTransmissionStatus(ctx context.Context) (*GuideDownloadTransmissionSnapshot, error) {
	return reader.transmission, reader.transmissionErr
}

type fakeGuideDownloadServicesWriter struct {
	validatePath string
	validateErr  error

	ensureDir string
	ensureErr error

	accessPath string
	canAccess  bool

	aria2Input       *GuideAria2InitInput
	writeAria2Err    error
	aria2Trackers    []string
	writeTrackersErr error
	restartAria2Err  error

	qbitInput      *GuideQbittorrentInitInput
	writeQbitErr   error
	restartQbitErr error

	transmissionInput      *GuideTransmissionInitInput
	writeTransmissionErr   error
	restartTransmissionErr error
}

func (writer *fakeGuideDownloadServicesWriter) ValidateDownloadPath(path string) error {
	writer.validatePath = path
	return writer.validateErr
}

func (writer *fakeGuideDownloadServicesWriter) EnsureDownloadDir(ctx context.Context, path string) error {
	writer.ensureDir = path
	return writer.ensureErr
}

func (writer *fakeGuideDownloadServicesWriter) CanAccessPath(path string) bool {
	writer.accessPath = path
	return writer.canAccess
}

func (writer *fakeGuideDownloadServicesWriter) WriteAria2Config(ctx context.Context, input GuideAria2InitInput) error {
	copied := input
	writer.aria2Input = &copied
	return writer.writeAria2Err
}

func (writer *fakeGuideDownloadServicesWriter) WriteAria2Trackers(ctx context.Context, trackers []string) error {
	writer.aria2Trackers = append([]string(nil), trackers...)
	return writer.writeTrackersErr
}

func (writer *fakeGuideDownloadServicesWriter) RestartAria2(ctx context.Context) error {
	return writer.restartAria2Err
}

func (writer *fakeGuideDownloadServicesWriter) WriteQbittorrentConfig(ctx context.Context, input GuideQbittorrentInitInput) error {
	copied := input
	writer.qbitInput = &copied
	return writer.writeQbitErr
}

func (writer *fakeGuideDownloadServicesWriter) RestartQbittorrent(ctx context.Context) error {
	return writer.restartQbitErr
}

func (writer *fakeGuideDownloadServicesWriter) WriteTransmissionConfig(ctx context.Context, input GuideTransmissionInitInput) error {
	copied := input
	writer.transmissionInput = &copied
	return writer.writeTransmissionErr
}

func (writer *fakeGuideDownloadServicesWriter) RestartTransmission(ctx context.Context) error {
	return writer.restartTransmissionErr
}

type fakeGuideDownloadServicesRuntime struct {
	trackers []string
	err      error
	rawInput string
}

func (runtime *fakeGuideDownloadServicesRuntime) ResolveAria2Trackers(ctx context.Context, rawTrackers string) ([]string, error) {
	runtime.rawInput = rawTrackers
	return append([]string(nil), runtime.trackers...), runtime.err
}

type fakeGuideAria2InitFacade struct {
	resp   *models.SDKNormalResponse
	err    error
	inputs []GuideAria2InitInput
}

func (facade *fakeGuideAria2InitFacade) InitAria2(ctx context.Context, input GuideAria2InitInput) (*models.SDKNormalResponse, error) {
	facade.inputs = append(facade.inputs, input)
	return facade.resp, facade.err
}

type fakeGuideQbittorrentInitFacade struct {
	resp   *models.SDKNormalResponse
	err    error
	inputs []GuideQbittorrentInitInput
}

func (facade *fakeGuideQbittorrentInitFacade) InitQbittorrent(ctx context.Context, input GuideQbittorrentInitInput) (*models.SDKNormalResponse, error) {
	facade.inputs = append(facade.inputs, input)
	return facade.resp, facade.err
}

type fakeGuideTransmissionInitFacade struct {
	resp   *models.SDKNormalResponse
	err    error
	inputs []GuideTransmissionInitInput
}

func (facade *fakeGuideTransmissionInitFacade) InitTransmission(ctx context.Context, input GuideTransmissionInitInput) (*models.SDKNormalResponse, error) {
	facade.inputs = append(facade.inputs, input)
	return facade.resp, facade.err
}

type fakeGuideDownloadServiceStatusFacade struct {
	resp     *models.GuideDownloadServiceResponse
	err      error
	getCalls int
}

func (facade *fakeGuideDownloadServiceStatusFacade) Get(ctx context.Context) (*models.GuideDownloadServiceResponse, error) {
	facade.getCalls++
	return facade.resp, facade.err
}

func TestGuideDownloadServiceStatusServiceBuildsLegacyResponseResult(t *testing.T) {
	service := GuideDownloadServiceStatusService{
		reader: &fakeGuideDownloadServicesReader{
			aria2: &GuideDownloadAria2Snapshot{
				Status:       "running",
				ConfigPath:   "/etc/aria2",
				DownloadPath: "/mnt/aria2",
				RPCPort:      6800,
				RPCToken:     "token",
				WebPath:      "/ariang",
			},
			qbittorrent: &GuideDownloadQbittorrentSnapshot{
				Status:       "stopped",
				ConfigPath:   "/etc/qbit",
				DownloadPath: "/mnt/qbit",
				WebPath:      ":8080",
			},
			transmission: &GuideDownloadTransmissionSnapshot{
				Status:       "not installed",
				ConfigPath:   "/etc/transmission",
				DownloadPath: "/mnt/trans",
				WebPath:      ":9091",
			},
		},
	}

	result, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected status service error: %v", err)
	}
	if result == nil || result.Result == nil {
		t.Fatalf("expected result payload, got %#v", result)
	}
	if result.Result.Aria2 == nil || result.Result.Aria2.Status != "running" || result.Result.Aria2.RPCPort != 6800 || result.Result.Aria2.WebPath != "/ariang" {
		t.Fatalf("unexpected aria2 result: %#v", result.Result.Aria2)
	}
	if result.Result.Qbittorrent == nil || result.Result.Qbittorrent.Status != "stopped" || result.Result.Qbittorrent.WebPath != ":8080" {
		t.Fatalf("unexpected qbittorrent result: %#v", result.Result.Qbittorrent)
	}
	if result.Result.Transmission == nil || result.Result.Transmission.Status != "not installed" || result.Result.Transmission.WebPath != ":9091" {
		t.Fatalf("unexpected transmission result: %#v", result.Result.Transmission)
	}
}

func TestGuideDownloadServiceStatusServicePreservesHardFailSemantics(t *testing.T) {
	aria2Err := errors.New("aria2 failed")
	service := GuideDownloadServiceStatusService{
		reader: &fakeGuideDownloadServicesReader{
			aria2Err: aria2Err,
		},
	}
	if _, err := service.Get(context.Background()); !errors.Is(err, aria2Err) {
		t.Fatalf("expected aria2 error, got %v", err)
	}

	qbitErr := errors.New("qbit failed")
	service.reader = &fakeGuideDownloadServicesReader{
		aria2:          &GuideDownloadAria2Snapshot{Status: "running"},
		qbittorrentErr: qbitErr,
	}
	if _, err := service.Get(context.Background()); !errors.Is(err, qbitErr) {
		t.Fatalf("expected qbittorrent error, got %v", err)
	}

	transErr := errors.New("transmission failed")
	service.reader = &fakeGuideDownloadServicesReader{
		aria2:           &GuideDownloadAria2Snapshot{Status: "running"},
		qbittorrent:     &GuideDownloadQbittorrentSnapshot{Status: "running"},
		transmissionErr: transErr,
	}
	if _, err := service.Get(context.Background()); !errors.Is(err, transErr) {
		t.Fatalf("expected transmission error, got %v", err)
	}
}

func TestBuildGuideDownloadServiceStatusResult(t *testing.T) {
	result := buildGuideDownloadServiceStatusResult(
		&GuideDownloadAria2Snapshot{Status: "running", ConfigPath: "/etc/a", DownloadPath: "/mnt/a", RPCPort: 6800, RPCToken: "x", WebPath: "/ariang"},
		&GuideDownloadQbittorrentSnapshot{Status: "stopped", ConfigPath: "/etc/q", DownloadPath: "/mnt/q", WebPath: ":8080"},
		&GuideDownloadTransmissionSnapshot{Status: "not installed", ConfigPath: "/etc/t", DownloadPath: "/mnt/t", WebPath: ":9091"},
	)
	if result == nil || result.Result == nil {
		t.Fatalf("expected response model, got %#v", result)
	}
	if result.Result.Aria2 == nil || result.Result.Qbittorrent == nil || result.Result.Transmission == nil {
		t.Fatalf("expected all service result sections, got %#v", result.Result)
	}
	var _ *models.GuideDownloadServiceResponse = result
}

func TestGuideAria2InitServicePreservesLegacyValidationAndTrackerSemantics(t *testing.T) {
	writer := &fakeGuideDownloadServicesWriter{canAccess: true}
	runtime := &fakeGuideDownloadServicesRuntime{trackers: []string{"udp://a", "udp://b"}}
	service := GuideAria2InitService{
		writer:  writer,
		runtime: runtime,
	}

	validateErr := errors.New("invalid path")
	writer.validateErr = validateErr
	if _, err := service.InitAria2(context.Background(), GuideAria2InitInput{DownloadPath: "/bad"}); !errors.Is(err, validateErr) {
		t.Fatalf("expected validate error, got %v", err)
	}

	writer.validateErr = nil
	writer.ensureErr = errors.New("mkdir failed")
	if _, err := service.InitAria2(context.Background(), GuideAria2InitInput{DownloadPath: "/mnt/data"}); err == nil || err.Error() != "/mnt/data 文件夹创建失败，请检查文件系统是否只读，或者已经存在同名文件" {
		t.Fatalf("unexpected ensure dir error: %v", err)
	}

	writer.ensureErr = nil
	writer.canAccess = false
	if _, err := service.InitAria2(context.Background(), GuideAria2InitInput{DownloadPath: "/mnt/data"}); err == nil || err.Error() != "无法访问下载路径" {
		t.Fatalf("unexpected access error: %v", err)
	}

	writer.canAccess = true
	runtime.err = errors.New("请求btTacker列表失败，请检查设备网络后，重试或手动配置")
	if _, err := service.InitAria2(context.Background(), GuideAria2InitInput{DownloadPath: "/mnt/data"}); err == nil || err.Error() != "请求btTacker列表失败，请检查设备网络后，重试或手动配置" {
		t.Fatalf("unexpected tracker error: %v", err)
	}

	runtime.err = nil
	writer.restartAria2Err = errors.New("restart failed")
	if _, err := service.InitAria2(context.Background(), GuideAria2InitInput{DownloadPath: "/mnt/data"}); err == nil || err.Error() != "aria2启动失败" {
		t.Fatalf("unexpected restart error: %v", err)
	}

	writer.restartAria2Err = nil
	resp, err := service.InitAria2(context.Background(), GuideAria2InitInput{
		BtTracker:    "udp://custom",
		ConfigPath:   "/etc/aria2",
		DownloadPath: "/mnt/data",
		RPCToken:     "token",
	})
	if err != nil {
		t.Fatalf("unexpected aria2 init error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected aria2 init response: %#v", resp)
	}
	if writer.validatePath != "/mnt/data" || writer.ensureDir != "/mnt/data" || writer.accessPath != "/mnt/data" {
		t.Fatalf("unexpected path handling: validate=%q ensure=%q access=%q", writer.validatePath, writer.ensureDir, writer.accessPath)
	}
	if writer.aria2Input == nil || writer.aria2Input.ConfigPath != "/etc/aria2" || writer.aria2Input.DownloadPath != "/mnt/data" || writer.aria2Input.RPCToken != "token" {
		t.Fatalf("unexpected aria2 input: %#v", writer.aria2Input)
	}
	if runtime.rawInput != "udp://custom" {
		t.Fatalf("unexpected tracker raw input: %q", runtime.rawInput)
	}
	if !reflect.DeepEqual(writer.aria2Trackers, []string{"udp://a", "udp://b"}) {
		t.Fatalf("unexpected tracker write: %v", writer.aria2Trackers)
	}
}

func TestServiceBackendGuideAria2InitCompatibility(t *testing.T) {
	orig := newGuideAria2InitServiceFacade
	defer func() { newGuideAria2InitServiceFacade = orig }()

	success := models.ResponseSuccess(0)
	facade := &fakeGuideAria2InitFacade{
		resp: &models.SDKNormalResponse{Success: &success},
	}
	newGuideAria2InitServiceFacade = func() guideAria2InitFacade { return facade }

	req := httptest.NewRequest("POST", "/guide/aria2", strings.NewReader(`{"btTracker":"udp://a","configPath":"/etc/aria2","downloadPath":"/mnt/data","rpcToken":"secret"}`))
	resp, err := (&ServiceBackend{}).PostGuideAria2Init(context.Background(), req)
	if err != nil || resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected PostGuideAria2Init response: resp=%#v err=%v", resp, err)
	}
	if len(facade.inputs) != 1 || facade.inputs[0] != (GuideAria2InitInput{
		BtTracker:    "udp://a",
		ConfigPath:   "/etc/aria2",
		DownloadPath: "/mnt/data",
		RPCToken:     "secret",
	}) {
		t.Fatalf("unexpected delegated aria2 input: %#v", facade.inputs)
	}
}

func TestServiceBackendGuideAria2InitCompatibilityPropagatesErrors(t *testing.T) {
	orig := newGuideAria2InitServiceFacade
	defer func() { newGuideAria2InitServiceFacade = orig }()

	serviceErr := errors.New("aria2 failed")
	newGuideAria2InitServiceFacade = func() guideAria2InitFacade {
		return &fakeGuideAria2InitFacade{err: serviceErr}
	}

	req := httptest.NewRequest("POST", "/guide/aria2", strings.NewReader(`{"configPath":"/etc/aria2","downloadPath":"/mnt/data","rpcToken":"secret"}`))
	if _, err := (&ServiceBackend{}).PostGuideAria2Init(context.Background(), req); !errors.Is(err, serviceErr) {
		t.Fatalf("expected PostGuideAria2Init error, got %v", err)
	}
}

func TestGuideQbittorrentInitServicePreservesLegacySemantics(t *testing.T) {
	writer := &fakeGuideDownloadServicesWriter{canAccess: true}
	service := GuideQbittorrentInitService{writer: writer}

	validateErr := errors.New("invalid path")
	writer.validateErr = validateErr
	if _, err := service.InitQbittorrent(context.Background(), GuideQbittorrentInitInput{DownloadPath: "/bad"}); !errors.Is(err, validateErr) {
		t.Fatalf("expected validate error, got %v", err)
	}

	writer.validateErr = nil
	writer.ensureErr = errors.New("mkdir failed")
	if _, err := service.InitQbittorrent(context.Background(), GuideQbittorrentInitInput{DownloadPath: "/mnt/qbit"}); err == nil || err.Error() != "/mnt/qbit 文件夹创建失败，请检查文件系统是否只读，或者已经存在同名文件" {
		t.Fatalf("unexpected ensure dir error: %v", err)
	}

	writer.ensureErr = nil
	writer.canAccess = false
	if _, err := service.InitQbittorrent(context.Background(), GuideQbittorrentInitInput{DownloadPath: "/mnt/qbit"}); err == nil || err.Error() != "无法访问下载路径" {
		t.Fatalf("unexpected access error: %v", err)
	}

	writer.canAccess = true
	writer.writeQbitErr = errors.New("write failed")
	if _, err := service.InitQbittorrent(context.Background(), GuideQbittorrentInitInput{DownloadPath: "/mnt/qbit"}); err == nil || err.Error() != "设置失败/mnt/qbit" {
		t.Fatalf("unexpected qbit write error: %v", err)
	}

	writer.writeQbitErr = nil
	writer.restartQbitErr = errors.New("restart failed")
	if _, err := service.InitQbittorrent(context.Background(), GuideQbittorrentInitInput{DownloadPath: "/mnt/qbit"}); err == nil || err.Error() != "启动失败" {
		t.Fatalf("unexpected qbit restart error: %v", err)
	}

	writer.restartQbitErr = nil
	resp, err := service.InitQbittorrent(context.Background(), GuideQbittorrentInitInput{
		ConfigPath:   "/etc/qbit",
		DownloadPath: "/mnt/qbit",
	})
	if err != nil {
		t.Fatalf("unexpected qbit init error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected qbit init response: %#v", resp)
	}
	if writer.qbitInput == nil || writer.qbitInput.ConfigPath != "/etc/qbit" || writer.qbitInput.DownloadPath != "/mnt/qbit" {
		t.Fatalf("unexpected qbit input: %#v", writer.qbitInput)
	}
}

func TestGuideTransmissionInitServicePreservesLegacySemantics(t *testing.T) {
	writer := &fakeGuideDownloadServicesWriter{canAccess: true}
	service := GuideTransmissionInitService{writer: writer}

	writer.writeTransmissionErr = errors.New("write failed")
	if _, err := service.InitTransmission(context.Background(), GuideTransmissionInitInput{DownloadPath: "/mnt/trans"}); err == nil || err.Error() != "设置失败/mnt/trans" {
		t.Fatalf("unexpected transmission write error: %v", err)
	}

	writer.writeTransmissionErr = nil
	writer.restartTransmissionErr = errors.New("restart failed")
	if _, err := service.InitTransmission(context.Background(), GuideTransmissionInitInput{DownloadPath: "/mnt/trans"}); err == nil || err.Error() != "启动失败" {
		t.Fatalf("unexpected transmission restart error: %v", err)
	}

	writer.restartTransmissionErr = nil
	resp, err := service.InitTransmission(context.Background(), GuideTransmissionInitInput{
		ConfigPath:   "/etc/transmission",
		DownloadPath: "/mnt/trans",
	})
	if err != nil {
		t.Fatalf("unexpected transmission init error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected transmission init response: %#v", resp)
	}
	if writer.transmissionInput == nil || writer.transmissionInput.ConfigPath != "/etc/transmission" || writer.transmissionInput.DownloadPath != "/mnt/trans" {
		t.Fatalf("unexpected transmission input: %#v", writer.transmissionInput)
	}
}

func TestServiceBackendGuideDownloadInitsCompatibility(t *testing.T) {
	origQbit := newGuideQbittorrentInitServiceFacade
	origTrans := newGuideTransmissionInitServiceFacade
	defer func() {
		newGuideQbittorrentInitServiceFacade = origQbit
		newGuideTransmissionInitServiceFacade = origTrans
	}()

	success := models.ResponseSuccess(0)
	qbitFacade := &fakeGuideQbittorrentInitFacade{resp: &models.SDKNormalResponse{Success: &success}}
	transFacade := &fakeGuideTransmissionInitFacade{resp: &models.SDKNormalResponse{Success: &success}}
	newGuideQbittorrentInitServiceFacade = func() guideQbittorrentInitFacade { return qbitFacade }
	newGuideTransmissionInitServiceFacade = func() guideTransmissionInitFacade { return transFacade }
	backend := &ServiceBackend{}

	qbitReq := httptest.NewRequest("POST", "/guide/qbit", strings.NewReader(`{"configPath":"/etc/qbit","downloadPath":"/mnt/qbit"}`))
	resp, err := backend.PostGuideQbittorrentInit(context.Background(), qbitReq)
	if err != nil || resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected PostGuideQbittorrentInit response: resp=%#v err=%v", resp, err)
	}
	if len(qbitFacade.inputs) != 1 || qbitFacade.inputs[0] != (GuideQbittorrentInitInput{ConfigPath: "/etc/qbit", DownloadPath: "/mnt/qbit"}) {
		t.Fatalf("unexpected qbit delegated input: %#v", qbitFacade.inputs)
	}

	transReq := httptest.NewRequest("POST", "/guide/transmission", strings.NewReader(`{"configPath":"/etc/transmission","downloadPath":"/mnt/trans"}`))
	resp, err = backend.PostGuideTransmissionInit(context.Background(), transReq)
	if err != nil || resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected PostGuideTransmissionInit response: resp=%#v err=%v", resp, err)
	}
	if len(transFacade.inputs) != 1 || transFacade.inputs[0] != (GuideTransmissionInitInput{ConfigPath: "/etc/transmission", DownloadPath: "/mnt/trans"}) {
		t.Fatalf("unexpected transmission delegated input: %#v", transFacade.inputs)
	}
}

func TestServiceBackendGuideDownloadInitsCompatibilityPropagateErrorsAndBodyDecodeWording(t *testing.T) {
	origQbit := newGuideQbittorrentInitServiceFacade
	origTrans := newGuideTransmissionInitServiceFacade
	defer func() {
		newGuideQbittorrentInitServiceFacade = origQbit
		newGuideTransmissionInitServiceFacade = origTrans
	}()

	serviceErr := errors.New("init failed")
	newGuideQbittorrentInitServiceFacade = func() guideQbittorrentInitFacade {
		return &fakeGuideQbittorrentInitFacade{err: serviceErr}
	}
	newGuideTransmissionInitServiceFacade = func() guideTransmissionInitFacade {
		return &fakeGuideTransmissionInitFacade{err: serviceErr}
	}
	backend := &ServiceBackend{}

	badQbitReq := httptest.NewRequest("POST", "/guide/qbit", strings.NewReader("{"))
	if _, err := backend.PostGuideQbittorrentInit(context.Background(), badQbitReq); err == nil || err.Error() != "获取请求数据失败" {
		t.Fatalf("expected qbit decode wording, got %v", err)
	}

	qbitReq := httptest.NewRequest("POST", "/guide/qbit", strings.NewReader(`{"configPath":"/etc/qbit","downloadPath":"/mnt/qbit"}`))
	if _, err := backend.PostGuideQbittorrentInit(context.Background(), qbitReq); !errors.Is(err, serviceErr) {
		t.Fatalf("expected qbit service error, got %v", err)
	}

	transReq := httptest.NewRequest("POST", "/guide/transmission", strings.NewReader(`{"configPath":"/etc/transmission","downloadPath":"/mnt/trans"}`))
	if _, err := backend.PostGuideTransmissionInit(context.Background(), transReq); !errors.Is(err, serviceErr) {
		t.Fatalf("expected transmission service error, got %v", err)
	}
}

func TestServiceBackendGuideDownloadServiceStatusCompatibility(t *testing.T) {
	orig := newGuideDownloadServiceStatusFacade
	defer func() { newGuideDownloadServiceStatusFacade = orig }()

	facade := &fakeGuideDownloadServiceStatusFacade{
		resp: &models.GuideDownloadServiceResponse{
			Result: &models.GuideDownloadServiceResponseResult{
				Aria2: &models.GuideDownloadAria2Info{Status: "running", RPCPort: 6800, WebPath: "/ariang"},
			},
		},
	}
	newGuideDownloadServiceStatusFacade = func() guideDownloadServiceStatusFacade { return facade }

	resp, err := (&ServiceBackend{}).GetGuideDownloadServiceStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected GetGuideDownloadServiceStatus error: %v", err)
	}
	if facade.getCalls != 1 {
		t.Fatalf("expected one facade Get call, got %d", facade.getCalls)
	}
	if resp == nil || resp.Result == nil || resp.Result.Aria2 == nil || resp.Result.Aria2.Status != "running" {
		t.Fatalf("unexpected GetGuideDownloadServiceStatus response: %#v", resp)
	}
}

func TestServiceBackendGuideDownloadServiceStatusCompatibilityPropagatesErrors(t *testing.T) {
	orig := newGuideDownloadServiceStatusFacade
	defer func() { newGuideDownloadServiceStatusFacade = orig }()

	serviceErr := errors.New("status failed")
	newGuideDownloadServiceStatusFacade = func() guideDownloadServiceStatusFacade {
		return &fakeGuideDownloadServiceStatusFacade{err: serviceErr}
	}

	if _, err := (&ServiceBackend{}).GetGuideDownloadServiceStatus(context.Background()); !errors.Is(err, serviceErr) {
		t.Fatalf("expected GetGuideDownloadServiceStatus error, got %v", err)
	}
}
