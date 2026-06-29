package service

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type DhcpConfigStore interface {
	LoadLanState(ctx context.Context) (*LanDhcpState, error)
	ApplyTagConfig(ctx context.Context, input DhcpTagConfigInput) error
	ApplyGatewayConfig(ctx context.Context, input DhcpGatewayInput, lanStatus LanStatusSnapshot) error
}

type LanStatusReader interface {
	ReadLanStatus(ctx context.Context) (LanStatusSnapshot, error)
}

type defaultDhcpConfigStore struct{}

type defaultLanStatusReader struct{}

type dhcpHostTagBinding struct {
	SectionName string
	TagName     string
}

var _ DhcpConfigStore = (*defaultDhcpConfigStore)(nil)
var _ LanStatusReader = (*defaultLanStatusReader)(nil)

func NewDefaultDhcpConfigStore() DhcpConfigStore {
	return &defaultDhcpConfigStore{}
}

func NewDefaultLanStatusReader() LanStatusReader {
	return &defaultLanStatusReader{}
}

func (store *defaultDhcpConfigStore) LoadLanState(ctx context.Context) (*LanDhcpState, error) {
	_ = ctx
	uci.LoadConfig("dhcp", true)
	uci.LoadConfig("floatip", true)

	lanDhcpOptions, _ := uci.Get("dhcp", "lan", "dhcp_option")
	dhcpIgnore := false
	ignoreValue, ok := uci.GetLast("dhcp", "lan", "ignore")
	if ok && ignoreValue == "1" {
		dhcpIgnore = true
	}

	state := &LanDhcpState{
		DhcpOptions: lanDhcpOptions,
		DhcpIgnore:  dhcpIgnore,
		Tags:        make([]DhcpTagRecord, 0),
	}

	floatEnabled, ok := uci.GetLast("floatip", "main", "enabled")
	if ok && floatEnabled == "1" {
		floatIP := &FloatIPSnapshot{Enabled: true}
		if setIP, ok := uci.GetLast("floatip", "main", "set_ip"); ok {
			floatIP.SetIP = strings.Split(setIP, "/")[0]
		}
		if checkIP, ok := uci.GetLast("floatip", "main", "check_ip"); ok {
			floatIP.CheckIP = checkIP
		}
		state.FloatIP = floatIP
	}

	tagSecs, ok := uci.GetSections("dhcp", "tag")
	if !ok {
		return state, nil
	}
	for _, name := range tagSecs {
		tag := DhcpTagRecord{TagName: name}
		tag.TagTitle, _ = uci.GetLast("dhcp", name, "tag_title")
		listValues, _ := uci.Get("dhcp", name, "dhcp_option")
		sort.Strings(listValues)
		tag.DhcpOption = listValues
		autoCreated, ok := uci.GetLast("dhcp", name, "AutoCreated")
		if ok {
			tag.AutoCreated, _ = strconv.ParseBool(autoCreated)
		}
		for _, value := range listValues {
			parts := splitDhcpOption(value)
			if len(parts) == 2 && parts[0] == "3" {
				tag.Gateway = parts[1]
			}
		}
		state.Tags = append(state.Tags, tag)
	}

	return state, nil
}

func (store *defaultDhcpConfigStore) ApplyTagConfig(ctx context.Context, input DhcpTagConfigInput) error {
	uci.LoadConfig("dhcp", true)
	hostBindings := make([]dhcpHostTagBinding, 0)
	hostSections, ok := uci.GetSections("dhcp", "host")
	if !ok {
		hostSections = []string{}
	}
	for _, hostSectionName := range hostSections {
		tag, _ := uci.GetLast("dhcp", hostSectionName, "tag")
		hostBindings = append(hostBindings, dhcpHostTagBinding{
			SectionName: hostSectionName,
			TagName:     tag,
		})
	}

	cmdList := buildDhcpTagCommands(input, hostBindings)

	if err := utils.UCIBatchRun(ctx, cmdList, "", 0); err != nil {
		return err
	}
	return utils.UciCommitAndApply(ctx, []string{"dhcp", "dnsmasq"})
}

func (store *defaultDhcpConfigStore) ApplyGatewayConfig(ctx context.Context, input DhcpGatewayInput, lanStatus LanStatusSnapshot) error {
	cmdList, err := buildDhcpGatewayCommands(input, lanStatus)
	if err != nil {
		return err
	}

	if err := utils.BatchRun(ctx, cmdList, 0); err != nil {
		return err
	}
	return utils.UciCommitAndApply(ctx, []string{"dhcp"})
}

func (reader *defaultLanStatusReader) ReadLanStatus(ctx context.Context) (LanStatusSnapshot, error) {
	lanStatus, err := ubusGetLanStatus(ctx)
	if err != nil {
		return LanStatusSnapshot{}, err
	}
	return LanStatusSnapshot{
		LanAddr:          lanStatus.lanAddr,
		Nexthop:          lanStatus.nexthop,
		IsDefaultGateway: lanStatus.isDefaultGateway,
	}, nil
}

func buildDhcpGatewayCommands(input DhcpGatewayInput, lanStatus LanStatusSnapshot) ([]string, error) {
	if input.DhcpGateway == "" {
		input.DhcpGateway = lanStatus.LanAddr
	}
	if input.DhcpGateway != "" && net.ParseIP(input.DhcpGateway) == nil {
		return nil, fmt.Errorf("dhcp gateway is not a valid IP address")
	}

	cmdList := make([]string, 0, 8)
	if input.DhcpEnabled {
		cmdList = append(cmdList, "uci del dhcp.lan.ignore")
		cmdList = append(cmdList, "uci del dhcp.lan.dhcp_option")
		cmdList = append(cmdList, "uci commit dhcp")
		if input.DhcpGateway != "" && input.DhcpGateway != lanStatus.LanAddr {
			cmdList = append(cmdList,
				fmt.Sprintf("uci add_list dhcp.lan.dhcp_option='3,%s'", input.DhcpGateway),
				fmt.Sprintf("uci add_list dhcp.lan.dhcp_option='6,%s'", input.DhcpGateway),
			)
		}
	} else {
		cmdList = append(cmdList, "uci del dhcp.lan.dhcp_option")
		cmdList = append(cmdList, "uci set dhcp.lan.ignore=1")
	}

	return cmdList, nil
}

func buildDhcpTagCommands(input DhcpTagConfigInput, hostBindings []dhcpHostTagBinding) []string {
	cmdList := make([]string, 0, len(input.DhcpOption)+len(hostBindings)+4)
	if input.Action == "modify" || input.Action == "delete" {
		cmdList = append(cmdList, fmt.Sprintf("del dhcp.%s", input.TagName))
	}

	if input.Action == "add" || input.Action == "modify" {
		cmdList = append(cmdList, fmt.Sprintf("set dhcp.%s=tag", input.TagName))
		cmdList = append(cmdList, fmt.Sprintf("set dhcp.%s.tag_title='%s'", input.TagName, input.TagTitle))
		for _, option := range input.DhcpOption {
			cmdList = append(cmdList, fmt.Sprintf("add_list dhcp.%s.dhcp_option='%s'", input.TagName, option))
		}
	}

	if input.Action == "modify" || input.Action == "delete" {
		for _, hostBinding := range hostBindings {
			if hostBinding.TagName == input.TagName {
				cmdList = append(cmdList, fmt.Sprintf("uci del dhcp.%s", hostBinding.SectionName))
			}
		}
	}

	return cmdList
}
