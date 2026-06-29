package netns

import (
	"errors"
	"reflect"
	"testing"
)

func TestCreateInterfaceCommands(t *testing.T) {
	t.Parallel()

	want := []string{
		"ip netns add dhns",
		"ip link add dheth0 type veth peer name dheth1",
		"ip link set dheth1 netns dhns",
		"ip link set dheth0 up",
		"echo OK",
	}
	if got := CreateInterfaceCommands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("CreateInterfaceCommands = %#v, want %#v", got, want)
	}
}

func TestValidateCreateInterfaceOutput(t *testing.T) {
	t.Parallel()

	if err := ValidateCreateInterfaceOutput([]byte("OK\n"), nil); err != nil {
		t.Fatalf("ValidateCreateInterfaceOutput returned error: %v", err)
	}

	readErr := errors.New("command failed")
	if err := ValidateCreateInterfaceOutput(nil, readErr); !errors.Is(err, readErr) {
		t.Fatalf("ValidateCreateInterfaceOutput error = %v, want %v", err, readErr)
	}

	if err := ValidateCreateInterfaceOutput([]byte(""), nil); err == nil || err.Error() != "unknown error" {
		t.Fatalf("ValidateCreateInterfaceOutput error = %v, want unknown error", err)
	}
}

func TestBridgePortCommands(t *testing.T) {
	t.Parallel()

	if got := BridgePortsQueryCommands(); !reflect.DeepEqual(got, []string{"uci get network.@device[0].ports"}) {
		t.Fatalf("BridgePortsQueryCommands = %#v", got)
	}

	wantAdd := []string{
		"uci add_list network.@device[0].ports=dheth0",
		"uci commit network",
		"/etc/init.d/network reload",
	}
	got, err := PlanBridgePortCommands([]byte("eth0 eth1"), nil)
	if err != nil {
		t.Fatalf("PlanBridgePortCommands returned error: %v", err)
	}
	if !reflect.DeepEqual(got, wantAdd) {
		t.Fatalf("PlanBridgePortCommands = %#v, want %#v", got, wantAdd)
	}

	got, err = PlanBridgePortCommands([]byte("eth0 dheth0 eth1"), nil)
	if err != nil {
		t.Fatalf("PlanBridgePortCommands returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("PlanBridgePortCommands = %#v, want empty", got)
	}

	readErr := errors.New("uci failed")
	if _, err := PlanBridgePortCommands(nil, readErr); !errors.Is(err, readErr) {
		t.Fatalf("PlanBridgePortCommands error = %v, want %v", err, readErr)
	}
}

func TestStartAndStopCommands(t *testing.T) {
	t.Parallel()

	wantStart := []string{
		"ip netns exec dhns killall startdhns",
		"ip netns exec dhns /usr/sbin/startdhns dhns &",
	}
	if got := StartCommands(); !reflect.DeepEqual(got, wantStart) {
		t.Fatalf("StartCommands = %#v, want %#v", got, wantStart)
	}

	wantStop := []string{
		"killall startdhns",
		"ip netns exec dhns ip addr flush dev dheth1",
	}
	if got := StopCommands(); !reflect.DeepEqual(got, wantStop) {
		t.Fatalf("StopCommands = %#v, want %#v", got, wantStop)
	}
}
