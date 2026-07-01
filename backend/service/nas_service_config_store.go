package service

import (
	"context"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
	"github.com/istoreos/quickstart/backend/utils"
)

var (
	loadNasServiceConfig = func(config string) {
		uci.LoadConfig(config, true)
	}
	getNasServiceLast = func(config string, section string, option string) (string, bool) {
		return uci.GetLast(config, section, option)
	}
	getNasServiceValues = func(config string, section string, option string) ([]string, bool) {
		return uci.Get(config, section, option)
	}
	getNasServiceSections = func(config string, sectionType string) ([]string, bool) {
		return uci.GetSections(config, sectionType)
	}
	readNasServiceNetworkStatus = func(ctx context.Context) (*models.NetworkStatusResponse, error) {
		return NetworkStatus(ctx, nil, false)
	}
	readNasServiceLinkeaseConfig = func(ctx context.Context, key string) ([]byte, error) {
		return utils.BatchOutputCmd(ctx, "uci get linkease.@linkease[0]."+key, 0)
	}
	hasNasServiceBinary = func(path string) bool {
		return Exists(path)
	}
)

type defaultNasServiceStatusReader struct{}

func newDefaultNasServiceStatusReader() NasServiceStatusReader {
	return defaultNasServiceStatusReader{}
}

func (defaultNasServiceStatusReader) ReadSambaShares() []*models.NasServiceSambaInfo {
	loadNasServiceConfig("unishare")

	sections, ok := getNasServiceSections("unishare", "share")
	if !ok {
		return nil
	}

	shares := make([]*models.NasServiceSambaInfo, 0, len(sections))
	for _, section := range sections {
		if !nasServiceHasProto("unishare", section, "samba") {
			continue
		}

		share := &models.NasServiceSambaInfo{}
		if value, ok := getNasServiceLast("unishare", section, "name"); ok {
			share.ShareName = value
		}
		if value, ok := getNasServiceLast("unishare", section, "path"); ok {
			share.Path = value
		}
		shares = append(shares, share)
	}
	return shares
}

func (defaultNasServiceStatusReader) ReadWebdavPort() (string, bool) {
	loadNasServiceConfig("unishare")
	return nasServiceReadWebdavPortFromUnishare()
}

func (defaultNasServiceStatusReader) ReadWebdavInfo() models.NasServiceWebdavInfo {
	loadNasServiceConfig("unishare")
	info := models.NasServiceWebdavInfo{}
	info.Port, _ = nasServiceReadWebdavPortFromUnishare()

	if sections, ok := getNasServiceSections("unishare", "share"); ok {
		for _, section := range sections {
			if !nasServiceHasProto("unishare", section, "webdav") {
				continue
			}
			if value, ok := getNasServiceLast("unishare", section, "path"); ok {
				info.Path = value
			}
			break
		}
	}
	return info
}

func nasServiceReadWebdavPortFromUnishare() (string, bool) {
	if value, ok := getNasServiceLast("unishare", "@global[0]", "webdav_port"); ok && len(value) > 0 {
		return value, true
	}
	return "8080", true
}

func nasServiceHasProto(config string, section string, proto string) bool {
	values, ok := getNasServiceValues(config, section, "proto")
	if !ok {
		return false
	}
	for _, value := range values {
		if value == proto {
			return true
		}
	}
	return false
}

func (defaultNasServiceStatusReader) ReadLinkeaseInfo(ctx context.Context) (bool, string, error) {
	enable, err := readNasServiceLinkeaseConfig(ctx, "preconfig")
	if err != nil {
		return false, "", nil
	}

	enabledByConfig := len(enable) > 10
	if !enabledByConfig {
		return false, "", nil
	}

	port, err := readNasServiceLinkeaseConfig(ctx, "port")
	if err != nil {
		return false, "", err
	}
	return true, strings.Trim(string(port), "\n"), nil
}

type defaultNasServiceRuntimeReader struct{}

func newDefaultNasServiceRuntimeReader() NasServiceRuntimeReader {
	return defaultNasServiceRuntimeReader{}
}

func (defaultNasServiceRuntimeReader) ReadLANIPv4(ctx context.Context) (string, error) {
	status, err := readNasServiceNetworkStatus(ctx)
	if err != nil {
		return "", err
	}
	if status == nil || status.Result == nil {
		return "", nil
	}
	return status.Result.Ipv4addr, nil
}

func (defaultNasServiceRuntimeReader) HasLinkeaseBinary() bool {
	return hasNasServiceBinary("/usr/sbin/linkease")
}
