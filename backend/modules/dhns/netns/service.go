package netns

import (
	"errors"
	"strings"
)

const VethHostDevice = "dheth0"

func CreateInterfaceCommands() []string {
	return []string{
		"ip netns add dhns",
		"ip link add dheth0 type veth peer name dheth1",
		"ip link set dheth1 netns dhns",
		"ip link set dheth0 up",
		"echo OK",
	}
}

func ValidateCreateInterfaceOutput(output []byte, err error) error {
	if err != nil {
		return err
	}
	if !strings.Contains(string(output), "OK") {
		return errors.New("unknown error")
	}
	return nil
}

func BridgePortsQueryCommands() []string {
	return []string{"uci get network.@device[0].ports"}
}

func PlanBridgePortCommands(output []byte, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}
	if strings.Contains(string(output), VethHostDevice) {
		return nil, nil
	}
	return []string{
		"uci add_list network.@device[0].ports=dheth0",
		"uci commit network",
		"/etc/init.d/network reload",
	}, nil
}

func StartCommands() []string {
	return []string{
		"ip netns exec dhns killall startdhns",
		"ip netns exec dhns /usr/sbin/startdhns dhns &",
	}
}

func StopCommands() []string {
	return []string{
		"killall startdhns",
		"ip netns exec dhns ip addr flush dev dheth1",
	}
}
