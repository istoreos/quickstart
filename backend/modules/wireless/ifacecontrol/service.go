package ifacecontrol

import "fmt"

type PlanInput struct {
	Enable         bool
	DeviceName     string
	IfaceName      string
	DeviceDisabled bool
	IfaceDisabled  bool
	ApplyCommand   string
}

type PlanResult struct {
	Commands []string
}

func Plan(input PlanInput) PlanResult {
	applyCommand := input.ApplyCommand
	if applyCommand == "" {
		applyCommand = "wifi"
	}
	oldDisabled := input.DeviceDisabled || input.IfaceDisabled
	if oldDisabled != input.Enable {
		return PlanResult{Commands: []string{applyCommand}}
	}
	if input.Enable {
		if input.DeviceDisabled {
			return PlanResult{Commands: []string{
				fmt.Sprintf(`uci set wireless.%s.disabled=0`, input.DeviceName),
				fmt.Sprintf(`uci set wireless.%s.disabled=0`, input.IfaceName),
				`uci commit wireless`,
				applyCommand,
			}}
		}
		return PlanResult{Commands: []string{
			fmt.Sprintf(`uci set wireless.%s.disabled=0`, input.IfaceName),
			`uci commit wireless`,
			applyCommand,
		}}
	}
	return PlanResult{Commands: []string{
		fmt.Sprintf(`uci set wireless.%s.disabled=1`, input.IfaceName),
		`uci commit wireless`,
		applyCommand,
	}}
}
