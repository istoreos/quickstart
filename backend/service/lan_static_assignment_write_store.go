package service

import (
	"context"
	"errors"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/modules/lancontrol/staticassignment"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type defaultStaticAssignmentWriteStore struct{}

var lanStaticAssignmentWriteLoadConfig = uci.LoadConfig
var lanStaticAssignmentWriteGetSections = uci.GetSections
var lanStaticAssignmentWriteGetLast = uci.GetLast
var lanStaticAssignmentWriteBatchRun = utils.BatchRun
var lanStaticAssignmentWriteCommitAndApply = utils.UciCommitAndApply

func NewDefaultStaticAssignmentWriteStore() StaticAssignmentWriteStore {
	return &defaultStaticAssignmentWriteStore{}
}

type defaultStaticAssignmentTagValidator struct {
	lanStatus LanStatusReader
	dhcpStore DhcpConfigStore
}

func NewDefaultStaticAssignmentTagValidator(lanStatus LanStatusReader, dhcpStore DhcpConfigStore) StaticAssignmentTagValidator {
	return &defaultStaticAssignmentTagValidator{
		lanStatus: lanStatus,
		dhcpStore: dhcpStore,
	}
}

func (store *defaultStaticAssignmentWriteStore) ApplyStaticAssignment(ctx context.Context, input StaticAssignmentWriteInput) error {
	_ = store

	_ = lanStaticAssignmentWriteLoadConfig("dhcp", true)
	sections, _ := lanStaticAssignmentWriteGetSections("dhcp", "host")
	hosts := buildStaticAssignmentHostRecords(sections)
	planInput := staticAssignmentPlanInput(input)

	if staticassignment.HasDuplicateIPConflict(planInput, hosts) {
		return errors.New("ip is already in use")
	}

	commands := staticassignment.BuildCommands(planInput, hosts, input.MaterializeAutoTag)
	if err := lanStaticAssignmentWriteBatchRun(ctx, commands, 0); err != nil {
		return err
	}
	return lanStaticAssignmentWriteCommitAndApply(ctx, []string{"dhcp"})
}

func buildStaticAssignmentHostRecords(sections []string) []staticassignment.HostRecord {
	hosts := make([]staticassignment.HostRecord, 0, len(sections))
	for _, sectionName := range sections {
		mac, ok := lanStaticAssignmentWriteGetLast("dhcp", sectionName, "mac")
		if !ok || mac == "" {
			continue
		}
		ip, _ := lanStaticAssignmentWriteGetLast("dhcp", sectionName, "ip")
		hosts = append(hosts, staticassignment.HostRecord{
			SectionName: sectionName,
			MAC:         mac,
			IP:          ip,
		})
	}
	return hosts
}

func staticAssignmentPlanInput(input StaticAssignmentWriteInput) staticassignment.Input {
	return staticassignment.Input{
		Action:      input.Action,
		AssignedMAC: input.AssignedMAC,
		AssignedIP:  input.AssignedIP,
		BindIP:      input.BindIP,
		Hostname:    input.Hostname,
		TagName:     input.TagName,
		TagTitle:    input.TagTitle,
	}
}

func (validator *defaultStaticAssignmentTagValidator) NormalizeTag(ctx context.Context, input StaticAssignmentWriteInput) (StaticAssignmentWriteInput, error) {
	if input.TagName == "" {
		return input, nil
	}

	lanStatus, err := validator.lanStatus.ReadLanStatus(ctx)
	if err != nil {
		return StaticAssignmentWriteInput{}, err
	}
	state, err := validator.dhcpStore.LoadLanState(ctx)
	if err != nil {
		return StaticAssignmentWriteInput{}, err
	}

	for _, tag := range buildGlobalDhcpTags(lanStatus, state) {
		if tag == nil || tag.TagName != input.TagName {
			continue
		}
		input.TagTitle = tag.TagTitle
		if tag.TagTitle == "default" {
			input.TagName = ""
			return input, nil
		}
		if tag.AutoCreated {
			input.MaterializeAutoTag = true
		}
		return input, nil
	}

	return StaticAssignmentWriteInput{}, errors.New("dhcp tag not found")
}
