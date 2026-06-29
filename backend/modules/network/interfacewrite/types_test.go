package interfacewrite

import (
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestBuildDeletePlan(t *testing.T) {
	t.Parallel()

	got := BuildDeletePlan(
		[]*models.NetworkInterfaceInfo{
			{Name: "lan"},
			{Name: "wan"},
			{Name: "wan6"},
		},
		[]*models.NetworkInterfaceConfig{
			{Name: "lan"},
			{Name: "wan"},
		},
	)

	want := []string{"wan6"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("delete plan = %#v, want %#v", got, want)
	}
}

func TestNormalizeDevices(t *testing.T) {
	t.Parallel()

	got := NormalizeDevices("lan", []string{"eth0"}, func(name string) bool {
		return name == "dsm-ext" || name == "vm-ext"
	})
	want := []string{"eth0", "dsm-ext", "vm-ext"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized devices = %#v, want %#v", got, want)
	}

	unchanged := NormalizeDevices("wan", []string{"eth1"}, func(name string) bool {
		return true
	})
	if !reflect.DeepEqual(unchanged, []string{"eth1"}) {
		t.Fatalf("non-lan devices should stay unchanged, got %#v", unchanged)
	}
}

func TestBuildBridgePlan(t *testing.T) {
	t.Parallel()

	createPlan := BuildBridgePlan(
		&models.NetworkInterfaceConfig{Name: "guest"},
		[]string{"eth0", "eth1"},
		nil,
	)
	wantCreate := []string{
		"uci add network device",
		"uci set network.@device[-1].type='bridge'",
		"uci set network.@device[-1].name=br-guest",
		"uci add_list network.@device[-1].ports=eth0",
		"uci add_list network.@device[-1].ports=eth1",
	}
	if !reflect.DeepEqual(createPlan, wantCreate) {
		t.Fatalf("create bridge plan = %#v, want %#v", createPlan, wantCreate)
	}

	updatePlan := BuildBridgePlan(
		&models.NetworkInterfaceConfig{Name: "lan"},
		[]string{"eth0", "dsm-ext"},
		[]DeviceSnapshot{
			{SectionName: "cfg01", Name: "br-lan"},
		},
	)
	wantUpdate := []string{
		"uci del network.cfg01.ports",
		"uci del network.cfg01.ports",
		"uci add_list network.cfg01.ports=eth0",
		"uci add_list network.cfg01.ports=dsm-ext",
	}
	if !reflect.DeepEqual(updatePlan, wantUpdate) {
		t.Fatalf("update bridge plan = %#v, want %#v", updatePlan, wantUpdate)
	}
}

func TestBuildInterfacePlan(t *testing.T) {
	t.Parallel()

	bridgePlan := BuildInterfacePlan(
		&models.NetworkInterfaceConfig{Name: "lan", Proto: "dhcp"},
		"br-lan",
		true,
	)
	wantBridge := []string{
		"uci set network.lan=interface",
		"uci set network.lan.proto=dhcp",
		"uci set network.lan.device=br-lan",
	}
	if !reflect.DeepEqual(bridgePlan, wantBridge) {
		t.Fatalf("bridge interface plan = %#v, want %#v", bridgePlan, wantBridge)
	}

	singlePlan := BuildInterfacePlan(
		&models.NetworkInterfaceConfig{Name: "wan", Proto: "pppoe"},
		"eth1",
		false,
	)
	wantSingle := []string{
		"uci set network.wan=interface",
		"uci set network.wan.proto=pppoe",
		"uci del network.wan.device",
		"uci set network.wan.device=eth1",
	}
	if !reflect.DeepEqual(singlePlan, wantSingle) {
		t.Fatalf("single interface plan = %#v, want %#v", singlePlan, wantSingle)
	}
}

func TestBuildFirewallBindingPlan(t *testing.T) {
	t.Parallel()

	got := BuildFirewallBindingPlan(
		&models.NetworkInterfaceConfig{Name: "guest", FirewallType: "lan"},
		[]FirewallZoneSnapshot{
			{SectionName: "cfg01", Name: "lan", Networks: []string{"lan"}},
			{SectionName: "cfg02", Name: "wan", Networks: []string{"wan"}},
		},
	)
	want := []string{"uci add_list firewall.cfg01.network=guest"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("firewall binding plan = %#v, want %#v", got, want)
	}

	existing := BuildFirewallBindingPlan(
		&models.NetworkInterfaceConfig{Name: "lan", FirewallType: "lan"},
		[]FirewallZoneSnapshot{
			{SectionName: "cfg01", Name: "lan", Networks: []string{"lan"}},
		},
	)
	if len(existing) != 0 {
		t.Fatalf("expected no firewall plan when binding exists, got %#v", existing)
	}
}
