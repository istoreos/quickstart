package service

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
	networkdevicelist "github.com/linkease/quick-start/istore-backend/modules/network/devicelist"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type networkDeviceListFacade interface {
	List(ctx context.Context) ([]*models.DeviceInfo, error)
}

var newNetworkDeviceListService = func() networkDeviceListFacade {
	return networkdevicelist.NewService(defaultNetworkDeviceListReader{})
}

type defaultNetworkDeviceListReader struct{}

func (reader defaultNetworkDeviceListReader) ReadLANInterfaceName(ctx context.Context) (string, error) {
	ifnameCmd := ". /lib/functions/network.sh; network_is_up \"$0\" || exit 1 ; network_get_device device \"$0\" ; echo \"$device\""
	ifnameB, err := exec.CommandContext(ctx, "sh", "-c", ifnameCmd, "lan").Output()
	if err != nil {
		return "", err
	}
	return strings.Replace(string(ifnameB), "\n", "", -1), nil
}

func (reader defaultNetworkDeviceListReader) ReadARPForInterface(ctx context.Context, ifname string) (string, error) {
	arpCMD := fmt.Sprintf("cat /proc/net/arp  | grep ' %v$'", ifname)
	ret, err := utils.BatchOutputCmd(ctx, arpCMD, 0)
	if err != nil {
		return "", err
	}
	return string(ret), nil
}
