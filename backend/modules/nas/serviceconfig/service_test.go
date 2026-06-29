package serviceconfig

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStatusReader struct {
	sambaShares  []*models.NasServiceSambaInfo
	webdavPort   string
	webdavOK     bool
	webdavInfo   models.NasServiceWebdavInfo
	linkeaseOn   bool
	linkeasePort string
	linkeaseErr  error
}

func (f fakeStatusReader) ReadSambaShares() []*models.NasServiceSambaInfo {
	return f.sambaShares
}

func (f fakeStatusReader) ReadWebdavPort() (string, bool) {
	return f.webdavPort, f.webdavOK
}

func (f fakeStatusReader) ReadWebdavInfo() models.NasServiceWebdavInfo {
	return f.webdavInfo
}

func (f fakeStatusReader) ReadLinkeaseInfo(ctx context.Context) (bool, string, error) {
	return f.linkeaseOn, f.linkeasePort, f.linkeaseErr
}

type fakeRuntimeReader struct {
	ipv4              string
	err               error
	hasLinkeaseBinary bool
}

func (f fakeRuntimeReader) ReadLANIPv4(ctx context.Context) (string, error) {
	return f.ipv4, f.err
}

func (f fakeRuntimeReader) HasLinkeaseBinary() bool {
	return f.hasLinkeaseBinary
}

type fakeConfigWriter struct {
	prepareSambaErr   error
	createSambaErr    error
	writeSambaErr     error
	writeWebdavErr    error
	restartWebdavErr  error
	prepareSambaCalls int
	createUserCalls   int
	writeSambaCalls   int
	writeWebdavCalls  int
	restartCalls      int
}

func (f *fakeConfigWriter) PrepareSamba(ctx context.Context) error {
	f.prepareSambaCalls++
	return f.prepareSambaErr
}

func (f *fakeConfigWriter) CreateSambaUser(ctx context.Context, username string, password string) error {
	f.createUserCalls++
	return f.createSambaErr
}

func (f *fakeConfigWriter) WriteSambaShare(ctx context.Context, input SambaCreateInput) error {
	f.writeSambaCalls++
	return f.writeSambaErr
}

func (f *fakeConfigWriter) WriteWebdavConfig(ctx context.Context, input WebdavCreateInput) error {
	f.writeWebdavCalls++
	return f.writeWebdavErr
}

func (f *fakeConfigWriter) RestartWebdav(ctx context.Context) error {
	f.restartCalls++
	return f.restartWebdavErr
}

type fakeTemplateWriter struct {
	err   error
	calls int
}

func (f *fakeTemplateWriter) EnableRoot() error {
	f.calls++
	return f.err
}

func TestStatusServiceAggregatesSambaWebdavAndLinkease(t *testing.T) {
	t.Parallel()

	service := NewStatusService(
		fakeStatusReader{
			sambaShares: []*models.NasServiceSambaInfo{{ShareName: "share", Path: "/mnt/data"}},
			webdavInfo: models.NasServiceWebdavInfo{
				Path:     "/mnt/data",
				Port:     "5244",
				Username: "user",
				Password: "pw",
			},
			linkeaseOn:   true,
			linkeasePort: "8897",
		},
		fakeRuntimeReader{hasLinkeaseBinary: true},
	)

	result, err := service.Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected status aggregation error: %v", err)
	}
	if len(result.Sambas) != 1 || result.Sambas[0].ShareName != "share" {
		t.Fatalf("unexpected samba result: %#v", result.Sambas)
	}
	if result.Webdav == nil || result.Webdav.Path != "/mnt/data" || result.Webdav.Port != "5244" || result.Webdav.Username != "user" || result.Webdav.Password != "pw" {
		t.Fatalf("unexpected webdav result: %#v", result.Webdav)
	}
	if result.Linkease == nil || !result.Linkease.Enabel || result.Linkease.Port != "8897" {
		t.Fatalf("unexpected linkease result: %#v", result.Linkease)
	}
}

func TestStatusServiceKeepsLinkeaseDisabledWithoutBinary(t *testing.T) {
	t.Parallel()

	service := NewStatusService(
		fakeStatusReader{
			webdavInfo:   models.NasServiceWebdavInfo{},
			linkeaseOn:   true,
			linkeasePort: "8897",
		},
		fakeRuntimeReader{hasLinkeaseBinary: false},
	)

	result, err := service.Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected status aggregation error: %v", err)
	}
	if result.Linkease == nil || result.Linkease.Enabel || result.Linkease.Port != "" {
		t.Fatalf("expected LinkEase disabled without binary, got %#v", result.Linkease)
	}
}

func TestStatusServicePropagatesReaderErrors(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("read linkease failed")
	service := NewStatusService(fakeStatusReader{linkeaseErr: expectedErr}, fakeRuntimeReader{})

	_, err := service.Read(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected reader error, got %v", err)
	}
}

func TestSambaCreateServiceRejectsDuplicateShare(t *testing.T) {
	t.Parallel()

	service := NewSambaCreateService(
		fakeStatusReader{sambaShares: []*models.NasServiceSambaInfo{{ShareName: "share"}}},
		fakeRuntimeReader{ipv4: "192.168.100.1"},
		&fakeConfigWriter{},
		&fakeTemplateWriter{},
	)

	_, err := service.Create(context.Background(), SambaCreateInput{
		ShareName: "share",
		RootPath:  "/mnt/data",
		Username:  "user",
		Password:  "pw",
	})
	if err == nil || err.Error() != "已存在同名samba共享" {
		t.Fatalf("expected duplicate-share error, got %v", err)
	}
}

func TestSambaCreateServiceMapsUserCreationError(t *testing.T) {
	t.Parallel()

	writer := &fakeConfigWriter{createSambaErr: errors.New("useradd failed")}
	templateWriter := &fakeTemplateWriter{}
	service := NewSambaCreateService(fakeStatusReader{}, fakeRuntimeReader{ipv4: "192.168.100.1"}, writer, templateWriter)

	_, err := service.Create(context.Background(), SambaCreateInput{
		ShareName: "share",
		RootPath:  "/mnt/data",
		Username:  "user",
		Password:  "pw",
	})
	expected := "添加samba用户失败，请修改用户名再试，注意不能包含大写字母，并且第一位不是数字"
	if err == nil || err.Error() != expected {
		t.Fatalf("expected legacy user-creation error, got %v", err)
	}
	if writer.prepareSambaCalls != 1 || templateWriter.calls != 1 || writer.createUserCalls != 1 {
		t.Fatalf("unexpected call sequence counts: prepare=%d template=%d create=%d", writer.prepareSambaCalls, templateWriter.calls, writer.createUserCalls)
	}
}

func TestWebdavCreateServiceWritesConfigAndBuildsURL(t *testing.T) {
	t.Parallel()

	writer := &fakeConfigWriter{}
	service := NewWebdavCreateService(fakeStatusReader{webdavPort: "5244", webdavOK: true}, fakeRuntimeReader{ipv4: "192.168.100.1"}, writer)

	result, err := service.Create(context.Background(), WebdavCreateInput{
		RootPath: "/mnt/data",
		Username: "user",
		Password: "pw",
	})
	if err != nil {
		t.Fatalf("unexpected WebDAV create error: %v", err)
	}
	if result == nil || result.Username != "user" || result.WebdavURL != "http://192.168.100.1:5244" {
		t.Fatalf("unexpected WebDAV result: %#v", result)
	}
	if writer.writeWebdavCalls != 1 || writer.restartCalls != 1 {
		t.Fatalf("unexpected WebDAV writer calls: write=%d restart=%d", writer.writeWebdavCalls, writer.restartCalls)
	}
}

func TestWebdavStatusServiceReadsConfiguredInfo(t *testing.T) {
	t.Parallel()

	service := NewWebdavStatusService(fakeStatusReader{
		webdavInfo: models.NasServiceWebdavInfo{
			Path:     "/mnt/data",
			Port:     "5244",
			Username: "user",
			Password: "pw",
		},
	})

	result, err := service.Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected WebDAV status error: %v", err)
	}
	if result == nil || result.Path != "/mnt/data" || result.Port != "5244" || result.Username != "user" || result.Password != "pw" {
		t.Fatalf("unexpected WebDAV status result: %#v", result)
	}
}

func TestSambaAndWebdavCreateServicesPropagateErrors(t *testing.T) {
	t.Parallel()

	runtimeErr := errors.New("network status failed")
	sambaService := NewSambaCreateService(fakeStatusReader{}, fakeRuntimeReader{err: runtimeErr}, &fakeConfigWriter{}, &fakeTemplateWriter{})
	if _, err := sambaService.Create(context.Background(), SambaCreateInput{
		ShareName: "share",
		RootPath:  "/mnt/data",
		Username:  "user",
		Password:  "pw",
	}); !errors.Is(err, runtimeErr) {
		t.Fatalf("expected samba runtime error, got %v", err)
	}

	writeErr := errors.New("write webdav failed")
	webdavService := NewWebdavCreateService(
		fakeStatusReader{webdavPort: "5244", webdavOK: true},
		fakeRuntimeReader{ipv4: "192.168.100.1"},
		&fakeConfigWriter{writeWebdavErr: writeErr},
	)
	if _, err := webdavService.Create(context.Background(), WebdavCreateInput{
		RootPath: "/mnt/data",
		Username: "user",
		Password: "pw",
	}); !errors.Is(err, writeErr) {
		t.Fatalf("expected webdav write error, got %v", err)
	}
}
