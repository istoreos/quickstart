package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/modules/lancontrol/floatgateway"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type LanFloatGatewayWriteStore interface {
	ReadState(ctx context.Context) (FloatGatewayStateSnapshot, []FloatGatewayDhcpTagSnapshot, []FloatGatewayDhcpHostSnapshot, error)
	ApplyPlan(ctx context.Context, plan FloatGatewayWriteExecutionPlan) error
}

type defaultLanFloatGatewayWriteStore struct{}

var lanFloatGatewayWriteExec = func(ctx context.Context, commands []string) error {
	return utils.BatchRun(ctx, commands, 0)
}

var lanFloatGatewayWriteLoadConfig = uci.LoadConfig

func NewDefaultLanFloatGatewayWriteStore() LanFloatGatewayWriteStore {
	return &defaultLanFloatGatewayWriteStore{}
}

func buildFloatGatewayWriteCommands(input FloatGatewayWriteInput) []string {
	return buildFloatGatewayConfigCommands(floatgateway.BuildConfig(input))
}

func buildFloatGatewayConfigCommands(config floatgateway.Config) []string {
	commands := []string{
		"uci del floatip.main",
		"uci commit floatip",
	}
	if config.Role == "" {
		return commands
	}

	enabled := 0
	if config.Enabled {
		enabled = 1
	}

	commands = append(commands,
		"uci set floatip.main=floatip",
		fmt.Sprintf("uci set floatip.main.enabled='%d'", enabled),
		fmt.Sprintf("uci set floatip.main.role='%s'", config.Role),
	)
	if config.UseScalarCheckIP {
		commands = append(commands, fmt.Sprintf("uci set floatip.main.check_ip='%s'", config.ScalarCheckIP))
	}
	if config.UseSetIP {
		commands = append(commands, fmt.Sprintf("uci set floatip.main.set_ip='%s'", config.SetIP))
	}
	if config.UseURLProbeSetting {
		commands = append(commands, fmt.Sprintf("uci set floatip.main.check_url='%s'", config.CheckURL))
		commands = append(commands, fmt.Sprintf("uci set floatip.main.check_url_timeout='%d'", config.CheckURLTimeout))
	}
	for _, checkIP := range config.CheckIPs {
		commands = append(commands, fmt.Sprintf("uci add_list floatip.main.check_ip='%s'", checkIP))
	}
	return commands
}

func shouldCleanupFloatGatewayDhcp(state FloatGatewayStateSnapshot, input FloatGatewayWriteInput) bool {
	return floatgateway.ShouldCleanupDhcp(state, input)
}

func buildFloatGatewayDhcpCleanupPlan(tags []FloatGatewayDhcpTagSnapshot, hosts []FloatGatewayDhcpHostSnapshot) FloatGatewayDhcpCleanupPlan {
	return floatgateway.BuildDhcpCleanupPlan(tags, hosts)
}

func buildFloatGatewayCleanupCommands(plan FloatGatewayDhcpCleanupPlan) []string {
	commands := make([]string, 0, len(plan.DeleteTagSections)+len(plan.DeleteHostSections))
	for _, section := range plan.DeleteTagSections {
		commands = append(commands, fmt.Sprintf("uci del dhcp.%s", section))
	}
	for _, section := range plan.DeleteHostSections {
		commands = append(commands, fmt.Sprintf("uci del dhcp.%s", section))
	}
	return commands
}

func (store *defaultLanFloatGatewayWriteStore) ReadState(ctx context.Context) (FloatGatewayStateSnapshot, []FloatGatewayDhcpTagSnapshot, []FloatGatewayDhcpHostSnapshot, error) {
	_ = store
	_ = ctx

	_ = lanFloatGatewayWriteLoadConfig("floatip", true)
	_ = lanFloatGatewayWriteLoadConfig("dhcp", true)

	enabledStr, _ := uci.GetLast("floatip", "main", "enabled")
	enabledInt, _ := strconv.Atoi(enabledStr)
	setIP, _ := uci.GetLast("floatip", "main", "set_ip")
	checkIP, _ := uci.GetLast("floatip", "main", "check_ip")

	state := FloatGatewayStateSnapshot{
		Enabled: enabledInt == 1,
		SetIP:   setIP,
		CheckIP: checkIP,
	}

	tagSections, ok := uci.GetSections("dhcp", "tag")
	if !ok {
		tagSections = []string{}
	}
	tags := make([]FloatGatewayDhcpTagSnapshot, 0, len(tagSections))
	for _, sectionName := range tagSections {
		tags = append(tags, FloatGatewayDhcpTagSnapshot{SectionName: sectionName})
	}

	hostSections, ok := uci.GetSections("dhcp", "host")
	if !ok {
		hostSections = []string{}
	}
	hosts := make([]FloatGatewayDhcpHostSnapshot, 0, len(hostSections))
	for _, sectionName := range hostSections {
		tag, _ := uci.GetLast("dhcp", sectionName, "tag")
		hosts = append(hosts, FloatGatewayDhcpHostSnapshot{
			SectionName: sectionName,
			Tag:         tag,
		})
	}

	return state, tags, hosts, nil
}

func (store *defaultLanFloatGatewayWriteStore) ApplyPlan(ctx context.Context, plan FloatGatewayWriteExecutionPlan) error {
	_ = store
	commands := append([]string{}, plan.FloatCommands...)
	commands = append(commands, buildFloatGatewayCleanupCommands(plan.CleanupPlan)...)
	return lanFloatGatewayWriteExec(ctx, commands)
}
