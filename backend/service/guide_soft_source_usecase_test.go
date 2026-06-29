package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeGuideSoftSourceReader struct {
	list       []*models.GuideSoftSourceInfo
	listErr    error
	current    *models.GuideSoftSourceInfo
	currentErr error
}

func (reader *fakeGuideSoftSourceReader) ListSources(ctx context.Context) ([]*models.GuideSoftSourceInfo, error) {
	return reader.list, reader.listErr
}

func (reader *fakeGuideSoftSourceReader) ReadCurrentSource(ctx context.Context) (*models.GuideSoftSourceInfo, error) {
	return reader.current, reader.currentErr
}

type fakeGuideSoftSourceWriter struct {
	replaced *models.GuideSoftSourceInfo
	err      error
}

func (writer *fakeGuideSoftSourceWriter) ReplaceSource(ctx context.Context, source models.GuideSoftSourceInfo) error {
	copied := source
	writer.replaced = &copied
	return writer.err
}

func TestGuideSoftSourceServiceListBuildsResponse(t *testing.T) {
	t.Parallel()

	service := GuideSoftSourceService{
		reader: &fakeGuideSoftSourceReader{
			list: []*models.GuideSoftSourceInfo{
				{Identity: "OpenWrtHttp", Name: "OpenWRT(HTTP)", URL: "http://downloads.openwrt.org/"},
			},
		},
		writer: &fakeGuideSoftSourceWriter{},
	}

	resp, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if resp == nil || len(resp.SoftSourceList) != 1 || resp.SoftSourceList[0].Identity != "OpenWrtHttp" {
		t.Fatalf("unexpected list response: %#v", resp)
	}
}

func TestGuideSoftSourceServiceGetBuildsResponse(t *testing.T) {
	t.Parallel()

	service := GuideSoftSourceService{
		reader: &fakeGuideSoftSourceReader{
			current: &models.GuideSoftSourceInfo{Identity: "USTC", Name: "中国科学技术大学", URL: "https://mirrors.ustc.edu.cn/openwrt/"},
		},
		writer: &fakeGuideSoftSourceWriter{},
	}

	resp, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if resp == nil || resp.SoftSource == nil || resp.SoftSource.Identity != "USTC" {
		t.Fatalf("unexpected get response: %#v", resp)
	}
}

func TestGuideSoftSourceServiceSetRejectsUnknownSource(t *testing.T) {
	t.Parallel()

	service := GuideSoftSourceService{
		reader: &fakeGuideSoftSourceReader{},
		writer: &fakeGuideSoftSourceWriter{},
	}

	if _, err := service.Set(context.Background(), GuideSoftSourceInput{SoftSourceIdentity: "unknown"}); err == nil || err.Error() != "没有获取到对应的软件源" {
		t.Fatalf("unexpected unknown-source error: %v", err)
	}
}

func TestGuideSoftSourceServiceSetMapsWriterFailure(t *testing.T) {
	t.Parallel()

	service := GuideSoftSourceService{
		reader: &fakeGuideSoftSourceReader{},
		writer: &fakeGuideSoftSourceWriter{err: errors.New("replace failed")},
	}

	if _, err := service.Set(context.Background(), GuideSoftSourceInput{SoftSourceIdentity: "USTC"}); err == nil || err.Error() != "修改软件源失败" {
		t.Fatalf("unexpected replace error: %v", err)
	}
}

func TestGuideSoftSourceServiceSetReusesGetAfterReplace(t *testing.T) {
	t.Parallel()

	reader := &fakeGuideSoftSourceReader{
		current: &models.GuideSoftSourceInfo{Identity: "Alibaba Cloud", Name: "阿里云", URL: "https://mirrors.aliyun.com/openwrt/"},
	}
	writer := &fakeGuideSoftSourceWriter{}
	service := GuideSoftSourceService{
		reader: reader,
		writer: writer,
	}

	resp, err := service.Set(context.Background(), GuideSoftSourceInput{SoftSourceIdentity: "Alibaba Cloud"})
	if err != nil {
		t.Fatalf("unexpected set error: %v", err)
	}
	if writer.replaced == nil || writer.replaced.Identity != "Alibaba Cloud" {
		t.Fatalf("unexpected replaced source: %#v", writer.replaced)
	}
	if resp == nil || resp.SoftSource == nil || resp.SoftSource.Identity != "Alibaba Cloud" {
		t.Fatalf("unexpected set response: %#v", resp)
	}
}

func TestServiceBackendGetGuideSoftSourceListCompatibility(t *testing.T) {
	prev := guideSoftSourceList
	defer func() { guideSoftSourceList = prev }()

	expected := &models.GuideSoftSourceListResponseResult{
		SoftSourceList: []*models.GuideSoftSourceInfo{
			{Identity: "OpenWrtHttp", Name: "OpenWRT(HTTP)", URL: "http://downloads.openwrt.org/"},
		},
	}
	guideSoftSourceList = func(ctx context.Context) (*models.GuideSoftSourceListResponseResult, error) {
		return expected, nil
	}

	resp, err := (&ServiceBackend{}).GetGuideSoftSourceList(context.Background())
	if err != nil {
		t.Fatalf("unexpected list wrapper error: %v", err)
	}
	if resp == nil || resp.Result == nil || len(resp.Result.SoftSourceList) != 1 || resp.Result.SoftSourceList[0].Identity != "OpenWrtHttp" {
		t.Fatalf("unexpected list wrapper response: %#v", resp)
	}
}

func TestServiceBackendGetGuideSoftSourceCompatibility(t *testing.T) {
	prev := guideSoftSourceGet
	defer func() { guideSoftSourceGet = prev }()

	expectedSource := &models.GuideSoftSourceInfo{Identity: "USTC", Name: "中国科学技术大学", URL: "https://mirrors.ustc.edu.cn/openwrt/"}
	guideSoftSourceGet = func(ctx context.Context) (*models.GuideSoftSourceResponseResult, error) {
		return &models.GuideSoftSourceResponseResult{SoftSource: expectedSource}, nil
	}

	resp, err := (&ServiceBackend{}).GetGuideSoftSource(context.Background())
	if err != nil {
		t.Fatalf("unexpected get wrapper error: %v", err)
	}
	if resp == nil || resp.Result == nil || resp.Result.SoftSource == nil || resp.Result.SoftSource.Identity != "USTC" {
		t.Fatalf("unexpected get wrapper response: %#v", resp)
	}
}

func TestServiceBackendPostGuideSoftSourceCompatibility(t *testing.T) {
	prev := guideSoftSourceSet
	defer func() { guideSoftSourceSet = prev }()

	expectedSource := &models.GuideSoftSourceInfo{Identity: "Alibaba Cloud", Name: "阿里云", URL: "https://mirrors.aliyun.com/openwrt/"}
	var captured GuideSoftSourceInput
	guideSoftSourceSet = func(ctx context.Context, input GuideSoftSourceInput) (*models.GuideSoftSourceResponseResult, error) {
		captured = input
		return &models.GuideSoftSourceResponseResult{SoftSource: expectedSource}, nil
	}

	req, err := http.NewRequest(http.MethodPost, "/guide/soft-source", strings.NewReader(`{"softSourceIdentity":"Alibaba Cloud"}`))
	if err != nil {
		t.Fatalf("unexpected request build error: %v", err)
	}
	resp, err := (&ServiceBackend{}).PostGuideSoftSource(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected post wrapper error: %v", err)
	}
	if captured.SoftSourceIdentity != "Alibaba Cloud" {
		t.Fatalf("unexpected post wrapper input: %#v", captured)
	}
	if resp == nil || resp.Result == nil || resp.Result.SoftSource == nil || resp.Result.SoftSource.Identity != "Alibaba Cloud" {
		t.Fatalf("unexpected post wrapper response: %#v", resp)
	}
}

func TestServiceBackendPostGuideSoftSourceMapsRequestParseError(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/guide/soft-source", strings.NewReader("{"))
	if err != nil {
		t.Fatalf("unexpected request build error: %v", err)
	}
	if _, err := (&ServiceBackend{}).PostGuideSoftSource(context.Background(), req); err == nil || err.Error() != "请求解析失败" {
		t.Fatalf("unexpected request parse error: %v", err)
	}
}
