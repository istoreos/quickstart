package service

import (
	"context"
	"fmt"

	"github.com/linkease/quick-start/istore-backend/utils"
)

type GuideDdnstoEnableInput struct {
	Token string
}

type GuideDdnstoAddressInput struct {
	Address string
}

type GuideDDNSWriter interface {
	EnableDdnsto(ctx context.Context, input GuideDdnstoEnableInput) (string, error)
	UpdateDdnstoAddress(ctx context.Context, input GuideDdnstoAddressInput) (string, error)
	ApplyDDNSConfig(ctx context.Context, cmds []string) error
	StartDDNSService(ctx context.Context, configName string) error
}

var writeGuideDDNSBatchOutErr = func(ctx context.Context, cmds []string, timeout int) (string, string, error) {
	return utils.BatchOutErr(ctx, cmds, timeout)
}

var writeGuideDDNSBatchRun = func(ctx context.Context, cmds []string, timeout int) error {
	return utils.BatchRun(ctx, cmds, timeout)
}

type defaultGuideDDNSWriter struct{}

func newDefaultGuideDDNSWriter() *defaultGuideDDNSWriter {
	return &defaultGuideDDNSWriter{}
}

func (writer *defaultGuideDDNSWriter) EnableDdnsto(ctx context.Context, input GuideDdnstoEnableInput) (string, error) {
	cmds := []string{
		"uci set ddnsto.@ddnsto[0].enabled=1",
		fmt.Sprintf("uci set ddnsto.@ddnsto[0].token=%v", input.Token),
		"uci commit ddnsto",
		"/etc/init.d/ddnsto restart",
	}
	_, stderr, err := writeGuideDDNSBatchOutErr(ctx, cmds, 0)
	return stderr, err
}

func (writer *defaultGuideDDNSWriter) UpdateDdnstoAddress(ctx context.Context, input GuideDdnstoAddressInput) (string, error) {
	cmds := []string{
		fmt.Sprintf("uci set ddnsto.@ddnsto[0].address=%v", input.Address),
		"uci commit ddnsto",
	}
	_, stderr, err := writeGuideDDNSBatchOutErr(ctx, cmds, 0)
	return stderr, err
}

func (writer *defaultGuideDDNSWriter) ApplyDDNSConfig(ctx context.Context, cmds []string) error {
	return writeGuideDDNSBatchRun(ctx, cmds, 0)
}

func (writer *defaultGuideDDNSWriter) StartDDNSService(ctx context.Context, configName string) error {
	return writeGuideDDNSBatchRun(ctx, []string{
		fmt.Sprintf("/usr/lib/ddns/dynamic_dns_lucihelper.sh -S %v -- start", configName),
	}, 0)
}
