package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeGlobalConfigLanStatusReader struct {
	readLanStatusFn func(ctx context.Context) (LanStatusSnapshot, error)
}

func (reader *fakeGlobalConfigLanStatusReader) ReadLanStatus(ctx context.Context) (LanStatusSnapshot, error) {
	if reader.readLanStatusFn != nil {
		return reader.readLanStatusFn(ctx)
	}
	return LanStatusSnapshot{}, nil
}

type fakeLanDhcpStateReader struct {
	loadLanStateFn func(ctx context.Context) (*LanDhcpState, error)
}

func (reader *fakeLanDhcpStateReader) LoadLanState(ctx context.Context) (*LanDhcpState, error) {
	if reader.loadLanStateFn != nil {
		return reader.loadLanStateFn(ctx)
	}
	return nil, nil
}

type fakeGlobalConfigFloatIPReader struct {
	readFloatIPStatusFn func(ctx context.Context) (FloatIPStatus, error)
}

func (reader *fakeGlobalConfigFloatIPReader) ReadFloatIPStatus(ctx context.Context) (FloatIPStatus, error) {
	if reader.readFloatIPStatusFn != nil {
		return reader.readFloatIPStatusFn(ctx)
	}
	return FloatIPStatus{}, nil
}

type fakeGlobalConfigSpeedLimitReader struct {
	readSpeedLimitStatusFn func(ctx context.Context) (SpeedLimitStatus, error)
}

func (reader *fakeGlobalConfigSpeedLimitReader) ReadSpeedLimitStatus(ctx context.Context) (SpeedLimitStatus, error) {
	if reader.readSpeedLimitStatusFn != nil {
		return reader.readSpeedLimitStatusFn(ctx)
	}
	return SpeedLimitStatus{}, nil
}

type fakeLanGlobalConfigFacade struct {
	resp *models.LANCtrlGlobalConfigResponse
	err  error
}

func (svc *fakeLanGlobalConfigFacade) GetGlobalConfigs(ctx context.Context) (*models.LANCtrlGlobalConfigResponse, error) {
	_ = ctx
	return svc.resp, svc.err
}

func TestServiceBackendGetLanGlobalConfigsDelegatesToLanGlobalConfigService(t *testing.T) {
	original := newLanGlobalConfigService
	defer func() {
		newLanGlobalConfigService = original
	}()

	expected := &models.LANCtrlGlobalConfigResponse{
		Result: &models.LANCtrlGlobalConfig{},
	}
	newLanGlobalConfigService = func() lanGlobalConfigFacade {
		return &fakeLanGlobalConfigFacade{resp: expected}
	}

	resp, err := (&ServiceBackend{}).GetLanGlobalConfigs(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp != expected {
		t.Fatalf("resp = %#v, want %#v", resp, expected)
	}
}

func TestServiceBackendGetLanGlobalConfigsPropagatesError(t *testing.T) {
	original := newLanGlobalConfigService
	defer func() {
		newLanGlobalConfigService = original
	}()

	wantErr := errors.New("global config failed")
	newLanGlobalConfigService = func() lanGlobalConfigFacade {
		return &fakeLanGlobalConfigFacade{err: wantErr}
	}

	resp, err := (&ServiceBackend{}).GetLanGlobalConfigs(context.Background())
	if !errors.Is(err, wantErr) || resp != nil {
		t.Fatalf("resp=%#v err=%v, want %v", resp, err, wantErr)
	}
}

func TestLanGlobalConfigServiceBuildsParentGatewaySelection(t *testing.T) {
	t.Parallel()

	svc := &LanGlobalConfigService{
		LanStatusReader: &fakeGlobalConfigLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{
					LanAddr: "192.168.100.1",
					Nexthop: "192.168.100.254",
				}, nil
			},
		},
		DhcpStore: &fakeLanDhcpStateReader{
			loadLanStateFn: func(ctx context.Context) (*LanDhcpState, error) {
				_ = ctx
				return &LanDhcpState{
					DhcpOptions: []string{"3,192.168.100.254", "6,192.168.100.254"},
				}, nil
			},
		},
		FloatIPReader:    &fakeGlobalConfigFloatIPReader{},
		SpeedLimitReader: &fakeGlobalConfigSpeedLimitReader{},
	}

	resp, err := svc.GetGlobalConfigs(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Result == nil || resp.Result.DhcpGlobal == nil {
		t.Fatal("expected dhcp global config to be present")
	}
	if got := len(resp.Result.DhcpGlobal.GatewaySels); got != 2 {
		t.Fatalf("expected 2 gateway selections, got %d", got)
	}
	if got := resp.Result.DhcpGlobal.GatewaySels[1].Title; got != "parent" {
		t.Fatalf("expected second gateway selection title %q, got %q", "parent", got)
	}
}

func TestBuildDhcpGlobalConfigPreservesMyselfThenParentOrder(t *testing.T) {
	t.Parallel()

	config := buildDhcpGlobalConfig(
		LanStatusSnapshot{
			LanAddr: "192.168.100.1",
			Nexthop: "192.168.100.254",
		},
		DhcpTagPlan{
			DhcpEnabled: true,
			DhcpGateway: "192.168.100.254",
		},
	)

	if config == nil {
		t.Fatal("expected dhcp global config")
	}
	if got := len(config.GatewaySels); got != 2 {
		t.Fatalf("expected 2 gateway selections, got %d", got)
	}
	if got := config.GatewaySels[0].Title; got != "myself" {
		t.Fatalf("expected first gateway selection title %q, got %q", "myself", got)
	}
	if got := config.GatewaySels[1].Title; got != "parent" {
		t.Fatalf("expected second gateway selection title %q, got %q", "parent", got)
	}
}

func TestLanGlobalConfigServiceKeepsCustomGatewayOption(t *testing.T) {
	t.Parallel()

	svc := &LanGlobalConfigService{
		LanStatusReader: &fakeGlobalConfigLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{
					LanAddr: "192.168.100.1",
					Nexthop: "192.168.100.254",
				}, nil
			},
		},
		DhcpStore: &fakeLanDhcpStateReader{
			loadLanStateFn: func(ctx context.Context) (*LanDhcpState, error) {
				_ = ctx
				return &LanDhcpState{
					DhcpOptions: []string{"3,8.8.8.8", "6,8.8.8.8"},
				}, nil
			},
		},
		FloatIPReader:    &fakeGlobalConfigFloatIPReader{},
		SpeedLimitReader: &fakeGlobalConfigSpeedLimitReader{},
	}

	resp, err := svc.GetGlobalConfigs(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Result == nil || resp.Result.DhcpGlobal == nil {
		t.Fatal("expected dhcp global config to be present")
	}
	if got := len(resp.Result.DhcpGlobal.GatewaySels); got != 2 {
		t.Fatalf("expected 2 gateway selections, got %d", got)
	}
	if got := resp.Result.DhcpGlobal.GatewaySels[1].Gateway; got != "8.8.8.8" {
		t.Fatalf("expected second gateway selection gateway %q, got %q", "8.8.8.8", got)
	}
}

func TestLanGlobalConfigServiceAggregatesDhcpTagsFloatGatewayAndSpeedLimit(t *testing.T) {
	t.Parallel()

	svc := &LanGlobalConfigService{
		LanStatusReader: &fakeGlobalConfigLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{
					LanAddr:          "192.168.100.1",
					Nexthop:          "192.168.100.254",
					IsDefaultGateway: true,
				}, nil
			},
		},
		DhcpStore: &fakeLanDhcpStateReader{
			loadLanStateFn: func(ctx context.Context) (*LanDhcpState, error) {
				_ = ctx
				return &LanDhcpState{}, nil
			},
		},
		FloatIPReader: &fakeGlobalConfigFloatIPReader{
			readFloatIPStatusFn: func(ctx context.Context) (FloatIPStatus, error) {
				_ = ctx
				return FloatIPStatus{
					Installed:       true,
					Enabled:         true,
					Role:            "main",
					SetIP:           "192.168.100.2",
					CheckIP:         "1.1.1.1",
					CheckURL:        "https://example.com/check",
					CheckURLTimeout: 15,
				}, nil
			},
		},
		SpeedLimitReader: &fakeGlobalConfigSpeedLimitReader{
			readSpeedLimitStatusFn: func(ctx context.Context) (SpeedLimitStatus, error) {
				_ = ctx
				return SpeedLimitStatus{
					Installed:     true,
					Enabled:       true,
					UploadSpeed:   2048,
					DownloadSpeed: 8192,
				}, nil
			},
		},
	}

	resp, err := svc.GetGlobalConfigs(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Result == nil {
		t.Fatal("expected result to be present")
	}
	if got := len(resp.Result.DhcpTags); got != 2 {
		t.Fatalf("expected 2 dhcp tags, got %d", got)
	}
	if got := resp.Result.DhcpTags[0].TagTitle; got != "default" {
		t.Fatalf("expected first dhcp tag title %q, got %q", "default", got)
	}
	if got := resp.Result.DhcpTags[1].TagTitle; got != "parent" {
		t.Fatalf("expected second dhcp tag title %q, got %q", "parent", got)
	}
	if resp.Result.FloatGateway == nil {
		t.Fatal("expected float gateway to be present")
	}
	if got := resp.Result.FloatGateway.CheckURL; got != "https://example.com/check" {
		t.Fatalf("expected float gateway CheckURL %q, got %q", "https://example.com/check", got)
	}
	if got := resp.Result.FloatGateway.CheckURLTimeout; got != 15 {
		t.Fatalf("expected float gateway CheckURLTimeout %d, got %d", 15, got)
	}
	if resp.Result.SpeedLimit == nil {
		t.Fatal("expected speed limit to be present")
	}
	if got := resp.Result.SpeedLimit.UploadSpeed; got != 2048 {
		t.Fatalf("expected upload speed %d, got %d", 2048, got)
	}
	if got := resp.Result.SpeedLimit.DownloadSpeed; got != 8192 {
		t.Fatalf("expected download speed %d, got %d", 8192, got)
	}
}

func TestLanGlobalConfigServiceBuildsDisabledDhcpResponse(t *testing.T) {
	t.Parallel()

	svc := &LanGlobalConfigService{
		LanStatusReader: &fakeGlobalConfigLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{
					LanAddr: "192.168.100.1",
				}, nil
			},
		},
		DhcpStore: &fakeLanDhcpStateReader{
			loadLanStateFn: func(ctx context.Context) (*LanDhcpState, error) {
				_ = ctx
				return &LanDhcpState{
					DhcpIgnore: true,
				}, nil
			},
		},
		FloatIPReader: &fakeGlobalConfigFloatIPReader{
			readFloatIPStatusFn: func(ctx context.Context) (FloatIPStatus, error) {
				_ = ctx
				return FloatIPStatus{
					Installed: true,
					Enabled:   true,
					Role:      "main",
				}, nil
			},
		},
		SpeedLimitReader: &fakeGlobalConfigSpeedLimitReader{
			readSpeedLimitStatusFn: func(ctx context.Context) (SpeedLimitStatus, error) {
				_ = ctx
				return SpeedLimitStatus{
					Installed:     true,
					Enabled:       true,
					UploadSpeed:   2048,
					DownloadSpeed: 8192,
				}, nil
			},
		},
	}

	resp, err := svc.GetGlobalConfigs(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Result == nil || resp.Result.DhcpGlobal == nil {
		t.Fatal("expected dhcp global config to be present")
	}
	if got := resp.Result.DhcpGlobal.DhcpEnabled; got {
		t.Fatalf("expected dhcp enabled %t, got %t", false, got)
	}
	if resp.Result.FloatGateway == nil {
		t.Fatal("expected float gateway to be present")
	}
	if got := resp.Result.FloatGateway.Enabled; !got {
		t.Fatalf("expected float gateway enabled %t, got %t", true, got)
	}
	if resp.Result.SpeedLimit == nil {
		t.Fatal("expected speed limit to be present")
	}
	if got := resp.Result.SpeedLimit.Enabled; !got {
		t.Fatalf("expected speed limit enabled %t, got %t", true, got)
	}
}

func TestLanGlobalConfigServiceContinuesWhenDhcpStateLoadFails(t *testing.T) {
	t.Parallel()

	svc := &LanGlobalConfigService{
		LanStatusReader: &fakeGlobalConfigLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{
					LanAddr: "192.168.100.1",
				}, nil
			},
		},
		DhcpStore: &fakeLanDhcpStateReader{
			loadLanStateFn: func(ctx context.Context) (*LanDhcpState, error) {
				_ = ctx
				return nil, context.DeadlineExceeded
			},
		},
		FloatIPReader: &fakeGlobalConfigFloatIPReader{
			readFloatIPStatusFn: func(ctx context.Context) (FloatIPStatus, error) {
				_ = ctx
				return FloatIPStatus{
					Installed: true,
					Enabled:   true,
					Role:      "main",
				}, nil
			},
		},
		SpeedLimitReader: &fakeGlobalConfigSpeedLimitReader{
			readSpeedLimitStatusFn: func(ctx context.Context) (SpeedLimitStatus, error) {
				_ = ctx
				return SpeedLimitStatus{
					Installed:     true,
					Enabled:       true,
					UploadSpeed:   2048,
					DownloadSpeed: 8192,
				}, nil
			},
		},
	}

	resp, err := svc.GetGlobalConfigs(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Result == nil || resp.Result.DhcpGlobal == nil {
		t.Fatal("expected dhcp global config to be present")
	}
	if got := resp.Result.DhcpGlobal.DhcpEnabled; !got {
		t.Fatalf("expected dhcp enabled %t, got %t", true, got)
	}
	if resp.Result.FloatGateway == nil {
		t.Fatal("expected float gateway to be present")
	}
	if got := resp.Result.FloatGateway.Enabled; !got {
		t.Fatalf("expected float gateway enabled %t, got %t", true, got)
	}
	if resp.Result.SpeedLimit == nil {
		t.Fatal("expected speed limit to be present")
	}
	if got := resp.Result.SpeedLimit.Enabled; !got {
		t.Fatalf("expected speed limit enabled %t, got %t", true, got)
	}
}

func TestLanGlobalConfigServicePreservesPersistedAndFloatIPDhcpTags(t *testing.T) {
	t.Parallel()

	svc := &LanGlobalConfigService{
		LanStatusReader: &fakeGlobalConfigLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{
					LanAddr:          "192.168.100.1",
					Nexthop:          "192.168.100.254",
					IsDefaultGateway: true,
				}, nil
			},
		},
		DhcpStore: &fakeLanDhcpStateReader{
			loadLanStateFn: func(ctx context.Context) (*LanDhcpState, error) {
				_ = ctx
				return &LanDhcpState{
					Tags: []DhcpTagRecord{
						{
							TagName:    "custom_dns",
							TagTitle:   "custom",
							Gateway:    "9.9.9.9",
							DhcpOption: []string{"6,9.9.9.9"},
						},
					},
					FloatIP: &FloatIPSnapshot{
						Enabled: true,
						SetIP:   "192.168.100.2",
						CheckIP: "1.1.1.1",
					},
				}, nil
			},
		},
		FloatIPReader:    &fakeGlobalConfigFloatIPReader{},
		SpeedLimitReader: &fakeGlobalConfigSpeedLimitReader{},
	}

	resp, err := svc.GetGlobalConfigs(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Result == nil {
		t.Fatal("expected result to be present")
	}
	if got := len(resp.Result.DhcpTags); got != 5 {
		t.Fatalf("expected 5 dhcp tags, got %d", got)
	}
	assertHasDhcpTag(t, resp.Result.DhcpTags, "custom_dns", "custom", "9.9.9.9")
	assertHasDhcpTag(t, resp.Result.DhcpTags, "t_auto_c0a86402", "floatip", "192.168.100.2")
	assertHasDhcpTag(t, resp.Result.DhcpTags, "t_auto_1010101", "bypass", "1.1.1.1")
}

func assertHasDhcpTag(t *testing.T, dhcpTags []*models.LANCtrlDhcpTagInfo, wantName, wantTitle, wantGateway string) {
	t.Helper()

	for _, tag := range dhcpTags {
		if tag.TagName != wantName {
			continue
		}
		if tag.TagTitle != wantTitle {
			t.Fatalf("expected tag %q title %q, got %q", wantName, wantTitle, tag.TagTitle)
		}
		if tag.Gateway != wantGateway {
			t.Fatalf("expected tag %q gateway %q, got %q", wantName, wantGateway, tag.Gateway)
		}
		return
	}

	t.Fatalf("expected tag %q to be present", wantName)
}
