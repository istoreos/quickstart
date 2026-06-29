package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeGuideStatusReadsReader struct {
	ddnstoConfig       *GuideDdnstoStatusSnapshot
	ddnstoConfigErr    error
	ddnsStatus         *GuideDDNSStatusSnapshot
	ddnsStatusErr      error
	downloadPartitions []string
	downloadErr        error
}

func (reader *fakeGuideStatusReadsReader) ReadDdnstoConfig(ctx context.Context) (*GuideDdnstoStatusSnapshot, error) {
	return reader.ddnstoConfig, reader.ddnstoConfigErr
}

func (reader *fakeGuideStatusReadsReader) ReadDDNSStatus(ctx context.Context) (*GuideDDNSStatusSnapshot, error) {
	return reader.ddnsStatus, reader.ddnsStatusErr
}

func (reader *fakeGuideStatusReadsReader) ReadDownloadPartitions(ctx context.Context) ([]string, error) {
	return append([]string(nil), reader.downloadPartitions...), reader.downloadErr
}

func TestGuideDdnstoConfigServiceBuildsResponse(t *testing.T) {
	t.Parallel()

	service := GuideDdnstoConfigService{
		reader: &fakeGuideStatusReadsReader{
			ddnstoConfig: &GuideDdnstoStatusSnapshot{DeviceID: "device-123"},
		},
	}

	resp, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected ddnsto config error: %v", err)
	}
	if resp.NetAddr != "127.0.0.1" || resp.DeviceID != "device-123" {
		t.Fatalf("unexpected ddnsto config response: %#v", resp)
	}
}

func TestGuideDdnstoConfigServicePropagatesReaderError(t *testing.T) {
	t.Parallel()

	service := GuideDdnstoConfigService{
		reader: &fakeGuideStatusReadsReader{ddnstoConfigErr: errors.New("ddnsto failed")},
	}

	if _, err := service.Get(context.Background()); err == nil || err.Error() != "ddnsto failed" {
		t.Fatalf("unexpected ddnsto config error: %v", err)
	}
}

func TestGuideDDNSStatusServiceBuildsResponse(t *testing.T) {
	t.Parallel()

	service := GuideDDNSStatusService{
		reader: &fakeGuideStatusReadsReader{
			ddnsStatus: &GuideDDNSStatusSnapshot{
				IPV4Domain:   "Stopped",
				IPV6Domain:   "ipv6.example.com",
				DdnstoDomain: "https://demo.example.com:443",
			},
		},
	}

	resp, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected ddns status error: %v", err)
	}
	if resp.IPV4Domain != "Stopped" || resp.IPV6Domain != "ipv6.example.com" || resp.DdnstoDomain != "https://demo.example.com:443" {
		t.Fatalf("unexpected ddns status response: %#v", resp)
	}
}

func TestGuideDDNSStatusServicePropagatesReaderError(t *testing.T) {
	t.Parallel()

	service := GuideDDNSStatusService{
		reader: &fakeGuideStatusReadsReader{ddnsStatusErr: errors.New("ddns failed")},
	}

	if _, err := service.Get(context.Background()); err == nil || err.Error() != "ddns failed" {
		t.Fatalf("unexpected ddns status error: %v", err)
	}
}

func TestGuideDownloadPartitionListServiceBuildsResponse(t *testing.T) {
	t.Parallel()

	service := GuideDownloadPartitionListService{
		reader: &fakeGuideStatusReadsReader{
			downloadPartitions: []string{"/mnt/data1/download", "/mnt/data2/download"},
		},
	}

	resp, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected download partition error: %v", err)
	}
	if len(resp.PartitionList) != 2 || resp.PartitionList[0] != "/mnt/data1/download" || resp.PartitionList[1] != "/mnt/data2/download" {
		t.Fatalf("unexpected download partition response: %#v", resp)
	}
}

func TestGuideDownloadPartitionListServicePropagatesReaderError(t *testing.T) {
	t.Parallel()

	service := GuideDownloadPartitionListService{
		reader: &fakeGuideStatusReadsReader{downloadErr: errors.New("download failed")},
	}

	if _, err := service.Get(context.Background()); err == nil || err.Error() != "download failed" {
		t.Fatalf("unexpected download partition error: %v", err)
	}
}

func TestServiceBackendGetGuideDdnstoConfigCompatibility(t *testing.T) {
	prev := newGuideDdnstoConfigServiceFacade
	defer func() { newGuideDdnstoConfigServiceFacade = prev }()

	expected := &models.GuideDdnstoConfigResponseResult{
		NetAddr:  "127.0.0.1",
		DeviceID: "device-123",
	}
	newGuideDdnstoConfigServiceFacade = func() *GuideDdnstoConfigService {
		return &GuideDdnstoConfigService{
			reader: &fakeGuideStatusReadsReader{
				ddnstoConfig: &GuideDdnstoStatusSnapshot{DeviceID: "device-123"},
			},
		}
	}

	resp, err := (&ServiceBackend{}).GetGuideDdnstoConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected ddnsto wrapper error: %v", err)
	}
	if resp == nil || resp.Result == nil || resp.Result.NetAddr != expected.NetAddr || resp.Result.DeviceID != expected.DeviceID {
		t.Fatalf("unexpected ddnsto wrapper response: %#v", resp)
	}
}

func TestServiceBackendGetGuideDdnsCompatibility(t *testing.T) {
	prev := newGuideDDNSStatusServiceFacade
	defer func() { newGuideDDNSStatusServiceFacade = prev }()

	newGuideDDNSStatusServiceFacade = func() *GuideDDNSStatusService {
		return &GuideDDNSStatusService{
			reader: &fakeGuideStatusReadsReader{
				ddnsStatus: &GuideDDNSStatusSnapshot{
					DdnstoDomain: "https://demo.example.com:443",
					IPV4Domain:   "Stopped",
					IPV6Domain:   "ipv6.example.com",
				},
			},
		}
	}

	resp, err := (&ServiceBackend{}).GetGuideDdns(context.Background())
	if err != nil {
		t.Fatalf("unexpected ddns wrapper error: %v", err)
	}
	if resp == nil || resp.Result == nil || resp.Result.DdnstoDomain != "https://demo.example.com:443" || resp.Result.IPV4Domain != "Stopped" || resp.Result.IPV6Domain != "ipv6.example.com" {
		t.Fatalf("unexpected ddns wrapper response: %#v", resp)
	}
}

func TestServiceBackendGetGuideDownloadPartitionListCompatibility(t *testing.T) {
	prev := newGuideDownloadPartitionListServiceFacade
	defer func() { newGuideDownloadPartitionListServiceFacade = prev }()

	newGuideDownloadPartitionListServiceFacade = func() *GuideDownloadPartitionListService {
		return &GuideDownloadPartitionListService{
			reader: &fakeGuideStatusReadsReader{
				downloadPartitions: []string{"/mnt/data1/download", "/mnt/data2/download"},
			},
		}
	}

	resp, err := (&ServiceBackend{}).GetGuideDownloadPartList(context.Background())
	if err != nil {
		t.Fatalf("unexpected download wrapper error: %v", err)
	}
	if resp == nil || resp.Result == nil || len(resp.Result.PartitionList) != 2 || resp.Result.PartitionList[0] != "/mnt/data1/download" || resp.Result.PartitionList[1] != "/mnt/data2/download" {
		t.Fatalf("unexpected download wrapper response: %#v", resp)
	}
}
