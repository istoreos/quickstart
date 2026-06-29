package ifacecontrol

import (
	"reflect"
	"testing"
)

func TestPlanRefreshesWhenRequestedStateAlreadyMatches(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name           string
		enable         bool
		deviceDisabled bool
		ifaceDisabled  bool
	}{
		{name: "already enabled", enable: true},
		{name: "already disabled by device", enable: false, deviceDisabled: true},
		{name: "already disabled by iface", enable: false, ifaceDisabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := Plan(PlanInput{
				Enable:         tc.enable,
				DeviceName:     "radio0",
				IfaceName:      "wifi2g",
				DeviceDisabled: tc.deviceDisabled,
				IfaceDisabled:  tc.ifaceDisabled,
			})
			if !reflect.DeepEqual(plan.Commands, []string{"wifi"}) {
				t.Fatalf("Commands = %#v, want refresh command", plan.Commands)
			}
		})
	}
}

func TestPlanEnableCommands(t *testing.T) {
	t.Parallel()

	plan := Plan(PlanInput{
		Enable:         true,
		DeviceName:     "radio0",
		IfaceName:      "wifi2g",
		DeviceDisabled: true,
		IfaceDisabled:  true,
	})

	assertCommands(t, plan.Commands, []string{
		`uci set wireless.radio0.disabled=0`,
		`uci set wireless.wifi2g.disabled=0`,
		`uci commit wireless`,
		`wifi`,
	})

	plan = Plan(PlanInput{
		Enable:        true,
		DeviceName:    "radio0",
		IfaceName:     "wifi2g",
		IfaceDisabled: true,
	})
	assertCommands(t, plan.Commands, []string{
		`uci set wireless.wifi2g.disabled=0`,
		`uci commit wireless`,
		`wifi`,
	})
}

func TestPlanDisableCommands(t *testing.T) {
	t.Parallel()

	plan := Plan(PlanInput{
		Enable:     false,
		DeviceName: "radio0",
		IfaceName:  "wifi2g",
	})

	assertCommands(t, plan.Commands, []string{
		`uci set wireless.wifi2g.disabled=1`,
		`uci commit wireless`,
		`wifi`,
	})
}

func assertCommands(t *testing.T, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Commands = %#v, want %#v", got, want)
	}
}
