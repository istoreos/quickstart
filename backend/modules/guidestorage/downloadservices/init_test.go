package downloadservices

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeWriter struct {
	validatePath string
	validateErr  error

	ensurePath string
	ensureErr  error

	accessPath string
	canAccess  bool

	aria2Input       *Aria2InitInput
	writeAria2Err    error
	aria2Trackers    []string
	writeTrackersErr error
	restartAria2Err  error

	qbitInput      *QbittorrentInitInput
	writeQbitErr   error
	restartQbitErr error

	transmissionInput      *TransmissionInitInput
	writeTransmissionErr   error
	restartTransmissionErr error
}

func (writer *fakeWriter) ValidateDownloadPath(path string) error {
	writer.validatePath = path
	return writer.validateErr
}

func (writer *fakeWriter) EnsureDownloadDir(ctx context.Context, path string) error {
	writer.ensurePath = path
	return writer.ensureErr
}

func (writer *fakeWriter) CanAccessPath(path string) bool {
	writer.accessPath = path
	return writer.canAccess
}

func (writer *fakeWriter) WriteAria2Config(ctx context.Context, input Aria2InitInput) error {
	copied := input
	writer.aria2Input = &copied
	return writer.writeAria2Err
}

func (writer *fakeWriter) WriteAria2Trackers(ctx context.Context, trackers []string) error {
	writer.aria2Trackers = append([]string(nil), trackers...)
	return writer.writeTrackersErr
}

func (writer *fakeWriter) RestartAria2(ctx context.Context) error {
	return writer.restartAria2Err
}

func (writer *fakeWriter) WriteQbittorrentConfig(ctx context.Context, input QbittorrentInitInput) error {
	copied := input
	writer.qbitInput = &copied
	return writer.writeQbitErr
}

func (writer *fakeWriter) RestartQbittorrent(ctx context.Context) error {
	return writer.restartQbitErr
}

func (writer *fakeWriter) WriteTransmissionConfig(ctx context.Context, input TransmissionInitInput) error {
	copied := input
	writer.transmissionInput = &copied
	return writer.writeTransmissionErr
}

func (writer *fakeWriter) RestartTransmission(ctx context.Context) error {
	return writer.restartTransmissionErr
}

type fakeRuntime struct {
	trackers []string
	err      error
	rawInput string
}

func (runtime *fakeRuntime) ResolveAria2Trackers(ctx context.Context, rawTrackers string) ([]string, error) {
	runtime.rawInput = rawTrackers
	return append([]string(nil), runtime.trackers...), runtime.err
}

func TestAria2InitServicePreservesLegacyFlow(t *testing.T) {
	writer := &fakeWriter{canAccess: true}
	runtime := &fakeRuntime{trackers: []string{"udp://a", "udp://b"}}
	service := NewAria2InitService(writer, runtime)
	input := Aria2InitInput{
		BtTracker:    "raw",
		ConfigPath:   "/etc/aria2",
		DownloadPath: "/mnt/aria2",
		RPCToken:     "token",
	}

	resp, err := service.InitAria2(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if writer.validatePath != "/mnt/aria2" || writer.ensurePath != "/mnt/aria2" || writer.accessPath != "/mnt/aria2" {
		t.Fatalf("unexpected path calls: validate=%q ensure=%q access=%q", writer.validatePath, writer.ensurePath, writer.accessPath)
	}
	if writer.aria2Input == nil || *writer.aria2Input != input {
		t.Fatalf("unexpected aria2 input: %#v", writer.aria2Input)
	}
	if runtime.rawInput != "raw" {
		t.Fatalf("expected raw tracker input, got %q", runtime.rawInput)
	}
	if !reflect.DeepEqual(writer.aria2Trackers, []string{"udp://a", "udp://b"}) {
		t.Fatalf("unexpected trackers: %#v", writer.aria2Trackers)
	}
}

func TestAria2InitServicePreservesLegacyErrors(t *testing.T) {
	validateErr := errors.New("invalid path")
	writer := &fakeWriter{validateErr: validateErr}
	service := NewAria2InitService(writer, &fakeRuntime{})
	if _, err := service.InitAria2(context.Background(), Aria2InitInput{DownloadPath: "/bad"}); !errors.Is(err, validateErr) {
		t.Fatalf("expected validation error, got %v", err)
	}

	writer = &fakeWriter{canAccess: true, ensureErr: errors.New("readonly")}
	service = NewAria2InitService(writer, &fakeRuntime{})
	if _, err := service.InitAria2(context.Background(), Aria2InitInput{DownloadPath: "/mnt/data"}); err == nil || err.Error() != "/mnt/data 文件夹创建失败，请检查文件系统是否只读，或者已经存在同名文件" {
		t.Fatalf("unexpected ensure dir error: %v", err)
	}

	writer = &fakeWriter{canAccess: false}
	service = NewAria2InitService(writer, &fakeRuntime{})
	if _, err := service.InitAria2(context.Background(), Aria2InitInput{DownloadPath: "/mnt/data"}); err == nil || err.Error() != "无法访问下载路径" {
		t.Fatalf("unexpected access error: %v", err)
	}

	runtimeErr := errors.New("tracker failed")
	writer = &fakeWriter{canAccess: true}
	service = NewAria2InitService(writer, &fakeRuntime{err: runtimeErr})
	if _, err := service.InitAria2(context.Background(), Aria2InitInput{DownloadPath: "/mnt/data"}); !errors.Is(err, runtimeErr) {
		t.Fatalf("expected tracker error, got %v", err)
	}

	writer = &fakeWriter{canAccess: true, restartAria2Err: errors.New("restart failed")}
	service = NewAria2InitService(writer, &fakeRuntime{})
	if _, err := service.InitAria2(context.Background(), Aria2InitInput{DownloadPath: "/mnt/data"}); err == nil || err.Error() != "aria2启动失败" {
		t.Fatalf("unexpected restart error: %v", err)
	}
}

func TestQbittorrentAndTransmissionInitServicesPreserveLegacyFlow(t *testing.T) {
	writer := &fakeWriter{canAccess: true}
	qbit := NewQbittorrentInitService(writer)
	qbitInput := QbittorrentInitInput{ConfigPath: "/etc/qbit", DownloadPath: "/mnt/qbit"}

	if _, err := qbit.InitQbittorrent(context.Background(), qbitInput); err != nil {
		t.Fatalf("unexpected qbittorrent error: %v", err)
	}
	if writer.qbitInput == nil || *writer.qbitInput != qbitInput {
		t.Fatalf("unexpected qbittorrent input: %#v", writer.qbitInput)
	}

	writer = &fakeWriter{canAccess: true}
	transmission := NewTransmissionInitService(writer)
	transmissionInput := TransmissionInitInput{ConfigPath: "/etc/transmission", DownloadPath: "/mnt/trans"}

	if _, err := transmission.InitTransmission(context.Background(), transmissionInput); err != nil {
		t.Fatalf("unexpected transmission error: %v", err)
	}
	if writer.transmissionInput == nil || *writer.transmissionInput != transmissionInput {
		t.Fatalf("unexpected transmission input: %#v", writer.transmissionInput)
	}
}

func TestQbittorrentAndTransmissionInitServicesPreserveLegacyErrors(t *testing.T) {
	writer := &fakeWriter{canAccess: true, writeQbitErr: errors.New("write failed")}
	qbit := NewQbittorrentInitService(writer)
	if _, err := qbit.InitQbittorrent(context.Background(), QbittorrentInitInput{DownloadPath: "/mnt/qbit"}); err == nil || err.Error() != "设置失败/mnt/qbit" {
		t.Fatalf("unexpected qbittorrent write error: %v", err)
	}

	writer = &fakeWriter{canAccess: true, restartQbitErr: errors.New("restart failed")}
	qbit = NewQbittorrentInitService(writer)
	if _, err := qbit.InitQbittorrent(context.Background(), QbittorrentInitInput{DownloadPath: "/mnt/qbit"}); err == nil || err.Error() != "启动失败" {
		t.Fatalf("unexpected qbittorrent restart error: %v", err)
	}

	writer = &fakeWriter{canAccess: true, writeTransmissionErr: errors.New("write failed")}
	transmission := NewTransmissionInitService(writer)
	if _, err := transmission.InitTransmission(context.Background(), TransmissionInitInput{DownloadPath: "/mnt/trans"}); err == nil || err.Error() != "设置失败/mnt/trans" {
		t.Fatalf("unexpected transmission write error: %v", err)
	}

	writer = &fakeWriter{canAccess: true, restartTransmissionErr: errors.New("restart failed")}
	transmission = NewTransmissionInitService(writer)
	if _, err := transmission.InitTransmission(context.Background(), TransmissionInitInput{DownloadPath: "/mnt/trans"}); err == nil || err.Error() != "启动失败" {
		t.Fatalf("unexpected transmission restart error: %v", err)
	}
}
