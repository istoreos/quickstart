package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStaticListLanStatusReader struct {
	readLanStatusFn func(ctx context.Context) (LanStatusSnapshot, error)
}

func (reader *fakeStaticListLanStatusReader) ReadLanStatus(ctx context.Context) (LanStatusSnapshot, error) {
	if reader.readLanStatusFn != nil {
		return reader.readLanStatusFn(ctx)
	}
	return LanStatusSnapshot{}, nil
}

type fakeLanStaticDeviceDhcpTagReader struct {
	readDhcpTagsFn func(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error)
}

func (reader *fakeLanStaticDeviceDhcpTagReader) ReadDhcpTags(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error) {
	if reader.readDhcpTagsFn != nil {
		return reader.readDhcpTagsFn(ctx, lanStatus)
	}
	return []*models.LANCtrlDhcpTagInfo{}, nil
}

type fakeStaticAssignmentListReader struct {
	readStaticAssignmentsFn func(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) ([]*models.LANStaticAssigned, error)
}

func (reader *fakeStaticAssignmentListReader) ReadStaticAssignments(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) ([]*models.LANStaticAssigned, error) {
	if reader.readStaticAssignmentsFn != nil {
		return reader.readStaticAssignmentsFn(ctx, tagList)
	}
	return []*models.LANStaticAssigned{}, nil
}

type fakeLanStaticDeviceListFacade struct {
	resp *models.LANCtrlStaticAssignedResponse
	err  error
}

func (svc *fakeLanStaticDeviceListFacade) GetListStaticDevices(ctx context.Context) (*models.LANCtrlStaticAssignedResponse, error) {
	_ = ctx
	return svc.resp, svc.err
}

func TestServiceBackendGetLanListStaticDevicesDelegatesToLanStaticDeviceListService(t *testing.T) {
	original := newLanStaticDeviceListService
	defer func() {
		newLanStaticDeviceListService = original
	}()

	expected := &models.LANCtrlStaticAssignedResponse{
		Result: []*models.LANStaticAssigned{
			{AssignedIP: "192.168.100.10", AssignedMac: "AA:BB:CC:DD:EE:10"},
		},
	}
	newLanStaticDeviceListService = func() lanStaticDeviceListFacade {
		return &fakeLanStaticDeviceListFacade{resp: expected}
	}

	resp, err := (&ServiceBackend{}).GetLanListStaticDevices(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp != expected {
		t.Fatalf("resp = %#v, want %#v", resp, expected)
	}
}

func TestServiceBackendGetLanListStaticDevicesPropagatesError(t *testing.T) {
	original := newLanStaticDeviceListService
	defer func() {
		newLanStaticDeviceListService = original
	}()

	wantErr := errors.New("static list failed")
	newLanStaticDeviceListService = func() lanStaticDeviceListFacade {
		return &fakeLanStaticDeviceListFacade{err: wantErr}
	}

	resp, err := (&ServiceBackend{}).GetLanListStaticDevices(context.Background())
	if !errors.Is(err, wantErr) || resp != nil {
		t.Fatalf("resp=%#v err=%v, want %v", resp, err, wantErr)
	}
}

func TestLanStaticDeviceListServiceReturnsLanStatusError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("read lan status failed")
	svc := &LanStaticDeviceListService{
		LanStatusReader: &fakeStaticListLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{}, wantErr
			},
		},
		DhcpTagReader:          &fakeLanStaticDeviceDhcpTagReader{},
		StaticAssignmentReader: &fakeStaticAssignmentListReader{},
	}

	_, err := svc.GetListStaticDevices(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestLanStaticDeviceListServiceReturnsDhcpTagError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("read dhcp tags failed")
	svc := &LanStaticDeviceListService{
		LanStatusReader: &fakeStaticListLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{LanAddr: "192.168.100.1"}, nil
			},
		},
		DhcpTagReader: &fakeLanStaticDeviceDhcpTagReader{
			readDhcpTagsFn: func(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error) {
				_ = ctx
				if lanStatus.LanAddr != "192.168.100.1" {
					t.Fatalf("unexpected lan status: %+v", lanStatus)
				}
				return nil, wantErr
			},
		},
		StaticAssignmentReader: &fakeStaticAssignmentListReader{},
	}

	_, err := svc.GetListStaticDevices(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestLanStaticDeviceListServiceReturnsStaticAssignmentError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("read static assignments failed")
	svc := &LanStaticDeviceListService{
		LanStatusReader: &fakeStaticListLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{LanAddr: "192.168.100.1"}, nil
			},
		},
		DhcpTagReader: &fakeLanStaticDeviceDhcpTagReader{
			readDhcpTagsFn: func(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error) {
				_ = ctx
				_ = lanStatus
				return []*models.LANCtrlDhcpTagInfo{}, nil
			},
		},
		StaticAssignmentReader: &fakeStaticAssignmentListReader{
			readStaticAssignmentsFn: func(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) ([]*models.LANStaticAssigned, error) {
				_ = ctx
				if len(tagList) != 0 {
					t.Fatalf("expected empty tag list, got %+v", tagList)
				}
				return nil, wantErr
			},
		},
	}

	_, err := svc.GetListStaticDevices(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestLanStaticDeviceListServicePreservesLegacyFieldSemantics(t *testing.T) {
	t.Parallel()

	svc := &LanStaticDeviceListService{
		LanStatusReader: &fakeStaticListLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{LanAddr: "192.168.100.1"}, nil
			},
		},
		DhcpTagReader: &fakeLanStaticDeviceDhcpTagReader{
			readDhcpTagsFn: func(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error) {
				_ = ctx
				_ = lanStatus
				return []*models.LANCtrlDhcpTagInfo{
					{TagName: "guest", TagTitle: "Guest", Gateway: "192.168.100.254"},
				}, nil
			},
		},
		StaticAssignmentReader: &fakeStaticAssignmentListReader{
			readStaticAssignmentsFn: func(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) ([]*models.LANStaticAssigned, error) {
				_ = ctx
				if len(tagList) != 1 || tagList[0].TagName != "guest" {
					t.Fatalf("unexpected tag list: %+v", tagList)
				}
				return []*models.LANStaticAssigned{
					{
						AssignedMac: "AA:BB:CC:DD:EE:01",
						AssignedIP:  "192.168.100.10",
						BindIP:      true,
						Hostname:    "printer",
						TagName:     "guest",
						TagTitle:    "Guest",
						DhcpGateway: "192.168.100.254",
					},
					{
						AssignedMac: "AA:BB:CC:DD:EE:02",
						Hostname:    "fallback",
						TagTitle:    "default",
					},
				}, nil
			},
		},
	}

	resp, err := svc.GetListStaticDevices(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	if len(resp.Result) != 2 {
		t.Fatalf("expected 2 static assignments, got %d", len(resp.Result))
	}
	if got := resp.Result[0]; got.AssignedMac != "AA:BB:CC:DD:EE:01" || got.AssignedIP != "192.168.100.10" || !got.BindIP || got.Hostname != "printer" || got.TagName != "guest" || got.TagTitle != "Guest" || got.DhcpGateway != "192.168.100.254" {
		t.Fatalf("unexpected first assignment: %+v", got)
	}
	if got := resp.Result[1]; got.AssignedMac != "AA:BB:CC:DD:EE:02" || got.BindIP || got.Hostname != "fallback" || got.TagName != "" || got.TagTitle != "default" || got.DhcpGateway != "" {
		t.Fatalf("unexpected second assignment: %+v", got)
	}
}

func TestLanStaticDeviceListServiceAllowsEmptyDhcpTags(t *testing.T) {
	t.Parallel()

	svc := &LanStaticDeviceListService{
		LanStatusReader: &fakeStaticListLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{LanAddr: "192.168.100.1"}, nil
			},
		},
		DhcpTagReader: &fakeLanStaticDeviceDhcpTagReader{
			readDhcpTagsFn: func(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error) {
				_ = ctx
				_ = lanStatus
				return []*models.LANCtrlDhcpTagInfo{}, nil
			},
		},
		StaticAssignmentReader: &fakeStaticAssignmentListReader{
			readStaticAssignmentsFn: func(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) ([]*models.LANStaticAssigned, error) {
				_ = ctx
				if len(tagList) != 0 {
					t.Fatalf("expected empty tag list, got %+v", tagList)
				}
				return []*models.LANStaticAssigned{
					{
						AssignedMac: "AA:BB:CC:DD:EE:03",
						Hostname:    "untagged",
					},
				}, nil
			},
		},
	}

	resp, err := svc.GetListStaticDevices(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || len(resp.Result) != 1 {
		t.Fatalf("expected 1 static assignment, got %+v", resp)
	}
	if got := resp.Result[0]; got.AssignedMac != "AA:BB:CC:DD:EE:03" || got.Hostname != "untagged" {
		t.Fatalf("unexpected assignment: %+v", got)
	}
}
