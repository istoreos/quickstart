package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/modules/lancontrol/staticassignment"
)

const dhcpConfigPath = "/etc/config/dhcp"

type defaultStaticAssignmentWriteStore struct{}

type dhcpConfigSnapshot struct {
	Data []byte
	Mode os.FileMode
}

var lanStaticAssignmentWriteLoadConfig = uci.LoadConfig
var lanStaticAssignmentWriteGetSections = uci.GetSections
var lanStaticAssignmentWriteGetLast = uci.GetLast
var lanStaticAssignmentWriteSnapshot = snapshotDhcpConfig
var lanStaticAssignmentWritePreflight = preflightStaticAssignment
var lanStaticAssignmentWriteMutate = mutateStaticAssignmentConfig
var lanStaticAssignmentWriteReload = reloadAndVerifyDnsmasq
var lanStaticAssignmentWriteRestore = restoreDhcpConfig

func NewDefaultStaticAssignmentWriteStore() StaticAssignmentWriteStore {
	return &defaultStaticAssignmentWriteStore{}
}

type defaultStaticAssignmentTagValidator struct {
	lanStatus LanStatusReader
	dhcpStore DhcpConfigStore
}

func NewDefaultStaticAssignmentTagValidator(lanStatus LanStatusReader, dhcpStore DhcpConfigStore) StaticAssignmentTagValidator {
	return &defaultStaticAssignmentTagValidator{lanStatus: lanStatus, dhcpStore: dhcpStore}
}

func (store *defaultStaticAssignmentWriteStore) ApplyStaticAssignment(ctx context.Context, input StaticAssignmentWriteInput) error {
	_ = store
	if err := lanStaticAssignmentWriteLoadConfig("dhcp", true); err != nil {
		return fmt.Errorf("load DHCP configuration: %w", err)
	}
	sections, _ := lanStaticAssignmentWriteGetSections("dhcp", "host")
	hosts := buildStaticAssignmentHostRecords(sections)
	if staticassignment.HasDuplicateIPConflict(staticAssignmentPlanInput(input), hosts) {
		return errors.New("ip is already in use")
	}
	if err := lanStaticAssignmentWritePreflight(ctx, input); err != nil {
		return fmt.Errorf("dnsmasq preflight failed: %w", err)
	}
	snapshot, err := lanStaticAssignmentWriteSnapshot()
	if err != nil {
		return fmt.Errorf("snapshot DHCP configuration: %w", err)
	}
	if err := lanStaticAssignmentWriteMutate(input, hosts); err != nil {
		return rollbackStaticAssignment(ctx, snapshot, fmt.Errorf("write DHCP configuration: %w", err))
	}
	if err := lanStaticAssignmentWriteReload(ctx); err != nil {
		return rollbackStaticAssignment(ctx, snapshot, fmt.Errorf("reload dnsmasq: %w", err))
	}
	return nil
}

func rollbackStaticAssignment(ctx context.Context, snapshot dhcpConfigSnapshot, applyErr error) error {
	if err := lanStaticAssignmentWriteRestore(ctx, snapshot); err != nil {
		return fmt.Errorf("%v; rollback failed: %w", applyErr, err)
	}
	return applyErr
}

func snapshotDhcpConfig() (dhcpConfigSnapshot, error) {
	info, err := os.Stat(dhcpConfigPath)
	if err != nil {
		return dhcpConfigSnapshot{}, err
	}
	data, err := os.ReadFile(dhcpConfigPath)
	if err != nil {
		return dhcpConfigSnapshot{}, err
	}
	return dhcpConfigSnapshot{Data: data, Mode: info.Mode().Perm()}, nil
}

func restoreDhcpConfig(ctx context.Context, snapshot dhcpConfigSnapshot) error {
	tmp, err := os.CreateTemp(filepath.Dir(dhcpConfigPath), ".quickstart-dhcp-rollback-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(snapshot.Mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(snapshot.Data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dhcpConfigPath); err != nil {
		return err
	}
	return reloadAndVerifyDnsmasq(ctx)
}

func preflightStaticAssignment(ctx context.Context, input StaticAssignmentWriteInput) error {
	if input.Action == "delete" {
		return nil
	}
	parts := []string{input.AssignedMAC}
	if input.TagName != "" {
		parts = append(parts, "set:"+input.TagName)
	}
	if input.Hostname != "" {
		parts = append(parts, input.Hostname)
	}
	if input.BindIP && input.AssignedIP != "" {
		parts = append(parts, input.AssignedIP)
	}
	command := exec.CommandContext(ctx, "dnsmasq", "--test", "--conf-file=-")
	command.Stdin = strings.NewReader("port=0\ndhcp-host=" + strings.Join(parts, ",") + "\n")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func mutateStaticAssignmentConfig(input StaticAssignmentWriteInput, hosts []staticassignment.HostRecord) error {
	return mutateStaticAssignmentConfigAt(filepath.Dir(dhcpConfigPath), input, hosts)
}

func mutateStaticAssignmentConfigAt(configDir string, input StaticAssignmentWriteInput, hosts []staticassignment.HostRecord) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		return err
	}
	for _, host := range hosts {
		if normalizeInventoryMAC(host.MAC) == normalizeInventoryMAC(input.AssignedMAC) {
			tree.DelSection("dhcp", host.SectionName)
		}
	}
	if input.Action != "delete" {
		if input.MaterializeAutoTag && input.TagName != "" {
			if err := tree.AddSection("dhcp", input.TagName, "tag"); err != nil {
				return err
			}
		}
		sectionName := "quickstart_" + strings.ToLower(strings.ReplaceAll(input.AssignedMAC, ":", ""))
		tree.DelSection("dhcp", sectionName)
		if err := tree.AddSection("dhcp", sectionName, "host"); err != nil {
			return err
		}
		options := map[string]string{
			"enabled":   "1",
			"mac":       input.AssignedMAC,
			"tag":       input.TagName,
			"tag_title": input.TagTitle,
			"name":      input.Hostname,
		}
		if input.BindIP {
			options["ip"] = input.AssignedIP
		}
		for _, name := range []string{"enabled", "mac", "tag", "tag_title", "name", "ip"} {
			value := options[name]
			if value != "" && !tree.Set("dhcp", sectionName, name, value) {
				return fmt.Errorf("set DHCP host option %s", name)
			}
		}
	}
	return tree.Commit()
}

func reloadAndVerifyDnsmasq(ctx context.Context) error {
	command := exec.CommandContext(ctx, "/etc/init.d/dnsmasq", "reload")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	var lastOutput []byte
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		check := exec.CommandContext(ctx, "/etc/init.d/dnsmasq", "running")
		lastOutput, lastErr = check.CombinedOutput()
		if lastErr == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return fmt.Errorf("dnsmasq is not running: %s: %w", bytes.TrimSpace(lastOutput), lastErr)
}

func buildStaticAssignmentHostRecords(sections []string) []staticassignment.HostRecord {
	hosts := make([]staticassignment.HostRecord, 0, len(sections))
	for _, sectionName := range sections {
		mac, ok := lanStaticAssignmentWriteGetLast("dhcp", sectionName, "mac")
		if !ok || mac == "" {
			continue
		}
		ip, _ := lanStaticAssignmentWriteGetLast("dhcp", sectionName, "ip")
		hosts = append(hosts, staticassignment.HostRecord{SectionName: sectionName, MAC: mac, IP: ip})
	}
	return hosts
}

func staticAssignmentPlanInput(input StaticAssignmentWriteInput) staticassignment.Input {
	return staticassignment.Input{
		Action: input.Action, AssignedMAC: input.AssignedMAC, AssignedIP: input.AssignedIP,
		BindIP: input.BindIP, Hostname: input.Hostname, TagName: input.TagName, TagTitle: input.TagTitle,
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
