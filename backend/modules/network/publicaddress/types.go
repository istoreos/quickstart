package publicaddress

import (
	"errors"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Snapshot struct {
	IPv4 string
	IPv6 string
}

func selectNetworkPublicAddress(snapshot Snapshot, ipVersion string) (string, error) {
	switch ipVersion {
	case "ipv4":
		if snapshot.IPv4 == "" {
			return "", errors.New("没有获取到ipv4信息")
		}
		return snapshot.IPv4, nil
	case "ipv6":
		if snapshot.IPv6 == "" {
			return "", errors.New("没有获取到ipv6信息")
		}
		return snapshot.IPv6, nil
	default:
		return "", errors.New("IPVersion参数错误" + ipVersion)
	}
}

func buildNetworkPublicAddressResult(address string, isPublic bool) *models.NetworkCheckPublicNetResponse {
	result := &models.NetworkCheckPublicNetResponseResult{}
	if isPublic {
		result.Address = address
	}

	resp := &models.NetworkCheckPublicNetResponse{}
	resp.Result = result
	return resp
}
