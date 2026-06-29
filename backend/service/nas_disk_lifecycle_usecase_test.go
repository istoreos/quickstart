package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func httptestRequest(body string) *http.Request {
	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	return req
}

type fakeNasDiskLifecycleFacade struct {
	mountPartitionInput      NasDiskPartitionMountInput
	mountPartitionResult     *models.PartitionInfo
	mountPartitionErr        error
	formatByDevicePathInput  NasDiskFormatByDevicePathInput
	formatByDevicePathResult *models.PartitionInfo
	formatByDevicePathErr    error
	initDiskInput            NasDiskInitInput
	initDiskResult           *models.NasDiskInfo
	initDiskErr              error
	initDiskRestInput        NasDiskInitRestInput
	initDiskRestResult       *models.NasDiskInfo
	initDiskRestErr          error
	generateMountPointPath   string
	generateMountPointResult string
	generateMountPointErr    error
}

func (svc *fakeNasDiskLifecycleFacade) MountPartition(ctx context.Context, input NasDiskPartitionMountInput) (*models.PartitionInfo, error) {
	svc.mountPartitionInput = input
	return svc.mountPartitionResult, svc.mountPartitionErr
}

func (svc *fakeNasDiskLifecycleFacade) FormatByDevicePath(ctx context.Context, input NasDiskFormatByDevicePathInput) (*models.PartitionInfo, error) {
	svc.formatByDevicePathInput = input
	return svc.formatByDevicePathResult, svc.formatByDevicePathErr
}

func (svc *fakeNasDiskLifecycleFacade) InitDisk(ctx context.Context, input NasDiskInitInput) (*models.NasDiskInfo, error) {
	svc.initDiskInput = input
	return svc.initDiskResult, svc.initDiskErr
}

func (svc *fakeNasDiskLifecycleFacade) InitDiskRest(ctx context.Context, input NasDiskInitRestInput) (*models.NasDiskInfo, error) {
	svc.initDiskRestInput = input
	return svc.initDiskRestResult, svc.initDiskRestErr
}

func (svc *fakeNasDiskLifecycleFacade) GenerateMountPoint(ctx context.Context, path string) (string, error) {
	svc.generateMountPointPath = path
	return svc.generateMountPointResult, svc.generateMountPointErr
}

func TestNasDiskPartitionMountCompatibilityDelegatesToLifecycleService(t *testing.T) {
	original := newNasDiskLifecycleService
	defer func() { newNasDiskLifecycleService = original }()

	facade := &fakeNasDiskLifecycleFacade{
		mountPartitionResult: &models.PartitionInfo{Path: "/dev/sda1", UUID: "uuid-1", MountPoint: "/mnt/data_sda1"},
	}
	newNasDiskLifecycleService = func() nasDiskLifecycleFacade {
		return facade
	}

	req := httptestRequest(`{"uuid":"uuid-1","path":"/dev/sda1","mountPoint":" /mnt/data_sda1 "}`)
	resp, err := NasDiskPartitionMount(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected NasDiskPartitionMount error: %v", err)
	}
	if facade.mountPartitionInput.UUID != "uuid-1" || facade.mountPartitionInput.Path != "/dev/sda1" || facade.mountPartitionInput.MountPoint != "/mnt/data_sda1" {
		t.Fatalf("unexpected lifecycle input: %#v", facade.mountPartitionInput)
	}
	if resp == nil || resp.Result == nil || resp.Result.MountPoint != "/mnt/data_sda1" {
		t.Fatalf("unexpected wrapper response: %#v", resp)
	}
}

func TestNasDiskMountPointCompatibilityDelegatesToLifecycleService(t *testing.T) {
	original := newNasDiskLifecycleService
	defer func() { newNasDiskLifecycleService = original }()

	facade := &fakeNasDiskLifecycleFacade{generateMountPointResult: "/mnt/data_sda1"}
	newNasDiskLifecycleService = func() nasDiskLifecycleFacade {
		return facade
	}

	resp, err := NasDiskMountPoint(context.Background(), models.NasDiskMountPointRequest{Path: "/dev/sda1"})
	if err != nil {
		t.Fatalf("unexpected NasDiskMountPoint error: %v", err)
	}
	if facade.generateMountPointPath != "/dev/sda1" {
		t.Fatalf("GenerateMountPoint path = %q", facade.generateMountPointPath)
	}
	if resp == nil || resp.Result == nil || resp.Result.Mountpoint != "/mnt/data_sda1" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestNasDiskMountPointCompatibilityPropagatesServiceError(t *testing.T) {
	original := newNasDiskLifecycleService
	defer func() { newNasDiskLifecycleService = original }()

	facadeErr := errors.New("mountPoint生成失败")
	newNasDiskLifecycleService = func() nasDiskLifecycleFacade {
		return &fakeNasDiskLifecycleFacade{generateMountPointErr: facadeErr}
	}

	if _, err := NasDiskMountPoint(context.Background(), models.NasDiskMountPointRequest{Path: "/dev/sda1"}); !errors.Is(err, facadeErr) {
		t.Fatalf("expected mount point error, got %v", err)
	}
}

func TestNasDiskPartitionMountCompatibilityPropagatesServiceError(t *testing.T) {
	original := newNasDiskLifecycleService
	defer func() { newNasDiskLifecycleService = original }()

	facadeErr := errors.New("mount failed")
	newNasDiskLifecycleService = func() nasDiskLifecycleFacade {
		return &fakeNasDiskLifecycleFacade{mountPartitionErr: facadeErr}
	}

	req := httptestRequest(`{"uuid":"uuid-1","path":"/dev/sda1","mountPoint":"/mnt/data_sda1"}`)
	if _, err := NasDiskPartitionMount(context.Background(), req); !errors.Is(err, facadeErr) {
		t.Fatalf("expected lifecycle error, got %v", err)
	}
}

func TestNasDiskPartitionFormatCompatibilityDelegatesToLifecycleService(t *testing.T) {
	original := newNasDiskLifecycleService
	defer func() { newNasDiskLifecycleService = original }()

	facade := &fakeNasDiskLifecycleFacade{
		formatByDevicePathResult: &models.PartitionInfo{Path: "/dev/sda1", UUID: "uuid-1", MountPoint: "/mnt/data_sda1"},
	}
	newNasDiskLifecycleService = func() nasDiskLifecycleFacade {
		return facade
	}

	resp, err := NasDiskPartitionFormatByDevicePath(context.Background(), "/dev/sda1")
	if err != nil {
		t.Fatalf("unexpected format-by-path error: %v", err)
	}
	if facade.formatByDevicePathInput.DevicePath != "/dev/sda1" {
		t.Fatalf("unexpected format input: %#v", facade.formatByDevicePathInput)
	}
	if resp == nil || resp.Result == nil || resp.Result.Path != "/dev/sda1" {
		t.Fatalf("unexpected format-by-path response: %#v", resp)
	}

	req := httptestRequest(`{"path":"/dev/sda2"}`)
	if _, err := NasDiskPartitionFormat(context.Background(), req); err != nil {
		t.Fatalf("unexpected format wrapper error: %v", err)
	}
	if facade.formatByDevicePathInput.DevicePath != "/dev/sda2" {
		t.Fatalf("expected request path to pass through wrapper, got %#v", facade.formatByDevicePathInput)
	}
}

func TestNasDiskPartitionFormatCompatibilityPropagatesServiceError(t *testing.T) {
	original := newNasDiskLifecycleService
	defer func() { newNasDiskLifecycleService = original }()

	facadeErr := errors.New("format failed")
	newNasDiskLifecycleService = func() nasDiskLifecycleFacade {
		return &fakeNasDiskLifecycleFacade{formatByDevicePathErr: facadeErr}
	}

	if _, err := NasDiskPartitionFormatByDevicePath(context.Background(), "/dev/sda1"); !errors.Is(err, facadeErr) {
		t.Fatalf("expected lifecycle error, got %v", err)
	}
}

func TestNasDiskInitCompatibilityDelegatesToLifecycleService(t *testing.T) {
	original := newNasDiskLifecycleService
	defer func() { newNasDiskLifecycleService = original }()

	facade := &fakeNasDiskLifecycleFacade{
		initDiskResult: &models.NasDiskInfo{Name: "sda", Path: "/dev/sda"},
	}
	newNasDiskLifecycleService = func() nasDiskLifecycleFacade {
		return facade
	}

	req := httptestRequest(`{"name":"sda","path":"/dev/sda"}`)
	resp, err := NasDiskInit(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected NasDiskInit error: %v", err)
	}
	if facade.initDiskInput.Name != "sda" || facade.initDiskInput.Path != "/dev/sda" {
		t.Fatalf("unexpected init input: %#v", facade.initDiskInput)
	}
	if resp == nil || resp.Result == nil || resp.Result.Name != "sda" {
		t.Fatalf("unexpected init response: %#v", resp)
	}
}

func TestNasDiskInitCompatibilityPropagatesServiceError(t *testing.T) {
	original := newNasDiskLifecycleService
	defer func() { newNasDiskLifecycleService = original }()

	facadeErr := errors.New("init failed")
	newNasDiskLifecycleService = func() nasDiskLifecycleFacade {
		return &fakeNasDiskLifecycleFacade{initDiskErr: facadeErr}
	}

	req := httptestRequest(`{"name":"sda","path":"/dev/sda"}`)
	if _, err := NasDiskInit(context.Background(), req); !errors.Is(err, facadeErr) {
		t.Fatalf("expected init error, got %v", err)
	}
}

func TestNasDiskInitRestCompatibilityDelegatesToLifecycleService(t *testing.T) {
	original := newNasDiskLifecycleService
	defer func() { newNasDiskLifecycleService = original }()

	facade := &fakeNasDiskLifecycleFacade{
		initDiskRestResult: &models.NasDiskInfo{Name: "sdb", Path: "/dev/sdb"},
	}
	newNasDiskLifecycleService = func() nasDiskLifecycleFacade {
		return facade
	}

	req := httptestRequest(`{"name":"sdb","path":"/dev/sdb"}`)
	resp, err := NasDiskInitRest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected NasDiskInitRest error: %v", err)
	}
	if facade.initDiskRestInput.Name != "sdb" || facade.initDiskRestInput.Path != "/dev/sdb" {
		t.Fatalf("unexpected init-rest input: %#v", facade.initDiskRestInput)
	}
	if resp == nil || resp.Result == nil || resp.Result.Name != "sdb" {
		t.Fatalf("unexpected init-rest response: %#v", resp)
	}
}

func TestNasDiskInitRestCompatibilityPropagatesLegacyDecodeAndServiceErrors(t *testing.T) {
	original := newNasDiskLifecycleService
	defer func() { newNasDiskLifecycleService = original }()

	req := httptestRequest(`{`)
	if _, err := NasDiskInitRest(context.Background(), req); err == nil || err.Error() != "获取参数失败" {
		t.Fatalf("expected legacy decode error, got %v", err)
	}

	facadeErr := errors.New("init rest failed")
	newNasDiskLifecycleService = func() nasDiskLifecycleFacade {
		return &fakeNasDiskLifecycleFacade{initDiskRestErr: facadeErr}
	}
	req = httptestRequest(`{"name":"sdb","path":"/dev/sdb"}`)
	if _, err := NasDiskInitRest(context.Background(), req); !errors.Is(err, facadeErr) {
		t.Fatalf("expected init-rest error, got %v", err)
	}
}
