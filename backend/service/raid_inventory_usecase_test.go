package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func raidTestRequest(body string) *http.Request {
	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	return req
}

type fakeRaidInventoryFacade struct {
	listResult       []*models.NasDiskInfo
	listErr          error
	detailPath       string
	detailResult     string
	detailErr        error
	createListResult []*models.RaidMemberInfo
	createListErr    error
}

func (svc *fakeRaidInventoryFacade) List(ctx context.Context) ([]*models.NasDiskInfo, error) {
	return svc.listResult, svc.listErr
}

func (svc *fakeRaidInventoryFacade) Detail(ctx context.Context, path string) (string, error) {
	svc.detailPath = path
	return svc.detailResult, svc.detailErr
}

func (svc *fakeRaidInventoryFacade) CreateList(ctx context.Context) ([]*models.RaidMemberInfo, error) {
	return svc.createListResult, svc.createListErr
}

func TestRaidInventoryCompatibilityDelegatesListDetailAndCreateList(t *testing.T) {
	original := newRaidInventoryService
	defer func() { newRaidInventoryService = original }()

	facade := &fakeRaidInventoryFacade{
		listResult:       []*models.NasDiskInfo{{Name: "md0", Path: "/dev/md0"}},
		detailResult:     "mdadm detail",
		createListResult: []*models.RaidMemberInfo{{Name: "sda", Path: "/dev/sda", Model: "disk", SizeStr: "1 TiB"}},
	}
	newRaidInventoryService = func() raidInventoryFacade {
		return facade
	}

	listResp, err := RaidGetList(context.Background())
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if listResp == nil || listResp.Result == nil || len(listResp.Result.Disks) != 1 || listResp.Result.Disks[0].Name != "md0" {
		t.Fatalf("unexpected list response: %#v", listResp)
	}

	detailResp, err := RaidPostDetail(context.Background(), raidTestRequest(`{"path":"/dev/md0"}`))
	if err != nil {
		t.Fatalf("unexpected detail error: %v", err)
	}
	if facade.detailPath != "/dev/md0" {
		t.Fatalf("unexpected detail path: %q", facade.detailPath)
	}
	if detailResp == nil || detailResp.Result == nil || detailResp.Result.Detail != "mdadm detail" {
		t.Fatalf("unexpected detail response: %#v", detailResp)
	}

	createListResp, err := RaidGetCreateList(context.Background())
	if err != nil {
		t.Fatalf("unexpected create-list error: %v", err)
	}
	if createListResp == nil || createListResp.Result == nil || len(createListResp.Result.Members) != 1 || createListResp.Result.Members[0].Name != "sda" {
		t.Fatalf("unexpected create-list response: %#v", createListResp)
	}
}

func TestRaidInventoryCompatibilityPropagatesFacadeErrors(t *testing.T) {
	original := newRaidInventoryService
	defer func() { newRaidInventoryService = original }()

	expectedErr := errors.New("inventory failed")
	newRaidInventoryService = func() raidInventoryFacade {
		return &fakeRaidInventoryFacade{
			listErr:       expectedErr,
			detailErr:     expectedErr,
			createListErr: expectedErr,
		}
	}

	if _, err := RaidGetList(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("expected list error, got %v", err)
	}
	if _, err := RaidPostDetail(context.Background(), raidTestRequest(`{"path":"/dev/md0"}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("expected detail error, got %v", err)
	}
	if _, err := RaidGetCreateList(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("expected create-list error, got %v", err)
	}
}

func TestRaidInventoryCompatibilityKeepsDetailDecodeError(t *testing.T) {
	original := newRaidInventoryService
	defer func() { newRaidInventoryService = original }()

	newRaidInventoryService = func() raidInventoryFacade {
		return &fakeRaidInventoryFacade{}
	}
	if _, err := RaidPostDetail(context.Background(), raidTestRequest(`{`)); err == nil {
		t.Fatalf("expected detail decode error")
	}
}
