package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestDefaultGuideDownloadServicesWriterDelegatesPathPreparation(t *testing.T) {
	originalValidate := validateGuideDownloadServicePath
	originalEnsure := createGuideDownloadServiceDir
	originalAccess := accessGuideDownloadServicePath
	defer func() {
		validateGuideDownloadServicePath = originalValidate
		createGuideDownloadServiceDir = originalEnsure
		accessGuideDownloadServicePath = originalAccess
	}()

	var gotPath string
	validateGuideDownloadServicePath = func(path string) error {
		gotPath = path
		return nil
	}
	var gotDir string
	createGuideDownloadServiceDir = func(ctx context.Context, path string) error {
		gotDir = path
		return nil
	}
	var accessArg string
	accessGuideDownloadServicePath = func(path string) bool {
		accessArg = path
		return true
	}

	writer := newDefaultGuideDownloadServicesWriter()
	if err := writer.ValidateDownloadPath("/mnt/data"); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
	if err := writer.EnsureDownloadDir(context.Background(), "/mnt/data"); err != nil {
		t.Fatalf("unexpected ensure dir error: %v", err)
	}
	if !writer.CanAccessPath("/mnt/data") {
		t.Fatal("expected path to be accessible")
	}
	if gotPath != "/mnt/data" || gotDir != "/mnt/data" || accessArg != "/mnt/data" {
		t.Fatalf("unexpected path delegation: validate=%q dir=%q access=%q", gotPath, gotDir, accessArg)
	}
}

func TestDefaultGuideDownloadServicesWriterBuildsLegacyMutationCommands(t *testing.T) {
	originalApply := applyGuideDownloadServiceCommands
	defer func() {
		applyGuideDownloadServiceCommands = originalApply
	}()

	var gotCalls [][]string
	applyGuideDownloadServiceCommands = func(ctx context.Context, cmds []string) error {
		gotCalls = append(gotCalls, append([]string(nil), cmds...))
		return nil
	}

	writer := newDefaultGuideDownloadServicesWriter()
	if err := writer.WriteAria2Config(context.Background(), GuideAria2InitInput{
		ConfigPath:   "/etc/aria2",
		DownloadPath: "/mnt/download",
		RPCToken:     "secret",
	}); err != nil {
		t.Fatalf("unexpected aria2 config error: %v", err)
	}
	if err := writer.WriteQbittorrentConfig(context.Background(), GuideQbittorrentInitInput{
		ConfigPath:   "/etc/qbit",
		DownloadPath: "/mnt/qbit",
	}); err != nil {
		t.Fatalf("unexpected qbittorrent config error: %v", err)
	}
	if err := writer.WriteTransmissionConfig(context.Background(), GuideTransmissionInitInput{
		ConfigPath:   "/etc/transmission",
		DownloadPath: "/mnt/transmission",
	}); err != nil {
		t.Fatalf("unexpected transmission config error: %v", err)
	}
	if err := writer.WriteAria2Trackers(context.Background(), []string{"udp://a", "udp://b"}); err != nil {
		t.Fatalf("unexpected aria2 tracker error: %v", err)
	}
	if err := writer.RestartAria2(context.Background()); err != nil {
		t.Fatalf("unexpected aria2 restart error: %v", err)
	}
	if err := writer.RestartQbittorrent(context.Background()); err != nil {
		t.Fatalf("unexpected qbittorrent restart error: %v", err)
	}
	if err := writer.RestartTransmission(context.Background()); err != nil {
		t.Fatalf("unexpected transmission restart error: %v", err)
	}

	expectedCalls := [][]string{
		{
			"uci set aria2.main.rpc_auth_method='token'",
			"uci set aria2.main.config_dir='/etc/aria2'",
			"uci set aria2.main.dir='/mnt/download'",
			"uci set aria2.main.enabled='1'",
			"uci set aria2.main.rpc_secret='secret'",
		},
		{
			"uci set qbittorrent.main.profile='/etc/qbit'",
			"uci set qbittorrent.main.SavePath='/mnt/qbit'",
			"uci set qbittorrent.main.enabled=1",
		},
		{
			"uci set transmission.@transmission[0].config_dir='/etc/transmission'",
			"uci set transmission.@transmission[0].download_dir='/mnt/transmission'",
			"uci set transmission.@transmission[0].enabled=1",
		},
		{"uci delete aria2.main.bt_tracker"},
		{"uci add_list aria2.main.bt_tracker='udp://a'"},
		{"uci add_list aria2.main.bt_tracker='udp://b'"},
		{"uci commit aria2", "/etc/init.d/aria2 restart"},
		{"uci commit qbittorrent", "/etc/init.d/qbittorrent restart"},
		{"uci commit transmission", "/etc/init.d/transmission restart"},
	}
	if !reflect.DeepEqual(gotCalls, expectedCalls) {
		t.Fatalf("unexpected command calls: %#v", gotCalls)
	}
}

func TestDefaultGuideDownloadServicesWriterAndRuntimePropagateErrors(t *testing.T) {
	originalValidate := validateGuideDownloadServicePath
	originalApply := applyGuideDownloadServiceCommands
	originalFetchTrackers := fetchGuideDownloadServiceAria2Trackers
	defer func() {
		validateGuideDownloadServicePath = originalValidate
		applyGuideDownloadServiceCommands = originalApply
		fetchGuideDownloadServiceAria2Trackers = originalFetchTrackers
	}()

	validateErr := errors.New("invalid path")
	validateGuideDownloadServicePath = func(path string) error {
		return validateErr
	}
	writer := newDefaultGuideDownloadServicesWriter()
	if err := writer.ValidateDownloadPath("/bad"); !errors.Is(err, validateErr) {
		t.Fatalf("expected validate error, got %v", err)
	}

	applyErr := errors.New("apply failed")
	applyGuideDownloadServiceCommands = func(ctx context.Context, cmds []string) error {
		return applyErr
	}
	if err := writer.RestartAria2(context.Background()); !errors.Is(err, applyErr) {
		t.Fatalf("expected restart error, got %v", err)
	}

	fetchGuideDownloadServiceAria2Trackers = func(ctx context.Context) (string, error) {
		return "", errors.New("boom")
	}
	runtime := newDefaultGuideDownloadServicesRuntime()
	if _, err := runtime.ResolveAria2Trackers(context.Background(), ""); err == nil || err.Error() != "请求btTacker列表失败，请检查设备网络后，重试或手动配置" {
		t.Fatalf("unexpected tracker bootstrap error: %v", err)
	}
}

func TestBuildGuideDownloadServiceTrackersPreservesLegacyParsing(t *testing.T) {
	runtime := newDefaultGuideDownloadServicesRuntime()
	trackers, err := runtime.ResolveAria2Trackers(context.Background(), "udp://a,\n udp://b\r\n")
	if err != nil {
		t.Fatalf("unexpected tracker parse error: %v", err)
	}
	if !reflect.DeepEqual(trackers, []string{"udp://a", "udp://b"}) {
		t.Fatalf("unexpected trackers: %v", trackers)
	}
}
