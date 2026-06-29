package core

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeServiceStore struct {
	installed    bool
	installedErr error
	running      bool
	installRet   string
	installErr   error
	apps         []*models.AppInstalled
	appsErr      error

	installedNames []string
	runningNames   []string
	installNames   []string
}

func (store *fakeServiceStore) IsInstalled(ctx context.Context, name string) (bool, error) {
	store.installedNames = append(store.installedNames, name)
	return store.installed, store.installedErr
}

func (store *fakeServiceStore) IsRunning(ctx context.Context, name string) bool {
	store.runningNames = append(store.runningNames, name)
	return store.running
}

func (store *fakeServiceStore) Install(ctx context.Context, name string) (string, error) {
	store.installNames = append(store.installNames, name)
	return store.installRet, store.installErr
}

func (store *fakeServiceStore) InstalledList(ctx context.Context) ([]*models.AppInstalled, error) {
	return store.apps, store.appsErr
}

func TestServiceCheckBuildsLegacyStatuses(t *testing.T) {
	tests := []struct {
		name         string
		req          models.AppCheckRequest
		installed    bool
		running      bool
		wantStatus   string
		wantRunCheck bool
	}{
		{name: "uninstalled", req: models.AppCheckRequest{Name: "demo"}, wantStatus: "uninstalled"},
		{name: "installed", req: models.AppCheckRequest{Name: "demo"}, installed: true, wantStatus: "installed"},
		{name: "running", req: models.AppCheckRequest{Name: "demo", CheckRunning: true}, installed: true, running: true, wantStatus: "running", wantRunCheck: true},
		{name: "stopped", req: models.AppCheckRequest{Name: "demo", CheckRunning: true}, installed: true, running: false, wantStatus: "stopped", wantRunCheck: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeServiceStore{installed: tt.installed, running: tt.running}
			service := NewService(store)

			resp, err := service.Check(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("check: %v", err)
			}
			if resp.Result == nil || resp.Result.Name != "demo" || resp.Result.Status != tt.wantStatus {
				t.Fatalf("unexpected response: %#v", resp)
			}
			if !reflect.DeepEqual(store.installedNames, []string{"demo"}) {
				t.Fatalf("unexpected installed checks: %#v", store.installedNames)
			}
			if tt.wantRunCheck && !reflect.DeepEqual(store.runningNames, []string{"demo"}) {
				t.Fatalf("expected running check, got %#v", store.runningNames)
			}
			if !tt.wantRunCheck && len(store.runningNames) != 0 {
				t.Fatalf("did not expect running check, got %#v", store.runningNames)
			}
		})
	}
}

func TestServiceCheckPreservesLegacyInstallCheckError(t *testing.T) {
	service := NewService(&fakeServiceStore{installedErr: errors.New("opkg failed")})

	_, err := service.Check(context.Background(), models.AppCheckRequest{Name: "demo"})
	if err == nil || err.Error() != "检测demo失败" {
		t.Fatalf("expected legacy check error, got %v", err)
	}
}

func TestServiceInstallPreservesLegacyResponses(t *testing.T) {
	store := &fakeServiceStore{installRet: "installing"}
	service := NewService(store)

	resp, err := service.Install(context.Background(), models.AppInstallRequest{Name: "demo"})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if resp.Success == nil || *resp.Success != 0 || resp.Detail != "installing" {
		t.Fatalf("unexpected success response: %#v", resp)
	}
	if !reflect.DeepEqual(store.installNames, []string{"demo"}) {
		t.Fatalf("unexpected install names: %#v", store.installNames)
	}

	_, err = service.Install(context.Background(), models.AppInstallRequest{})
	if err == nil || err.Error() != "missing param" {
		t.Fatalf("expected missing param error, got %v", err)
	}

	service = NewService(&fakeServiceStore{installErr: errors.New("install failed")})
	resp, err = service.Install(context.Background(), models.AppInstallRequest{Name: "demo"})
	if err != nil {
		t.Fatalf("install errors should return SDK response, got %v", err)
	}
	if resp.Error == "" || resp.Scope != "1003" {
		t.Fatalf("unexpected install error response: %#v", resp)
	}
}

func TestServiceInstalledListDelegatesStore(t *testing.T) {
	want := []*models.AppInstalled{{Name: "demo", Title: "Demo"}}
	service := NewService(&fakeServiceStore{apps: want})

	got, err := service.InstalledList(context.Background())
	if err != nil {
		t.Fatalf("installed list: %v", err)
	}
	if !reflect.DeepEqual(got, models.AppInstalledListResponse(want)) {
		t.Fatalf("unexpected installed list\nwant: %#v\n got: %#v", want, got)
	}

	listErr := errors.New("list failed")
	service = NewService(&fakeServiceStore{appsErr: listErr})
	if _, err := service.InstalledList(context.Background()); !errors.Is(err, listErr) {
		t.Fatalf("expected list error, got %v", err)
	}
}
