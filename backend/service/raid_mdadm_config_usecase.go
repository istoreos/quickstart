package service

import (
	"context"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/modules/raid/mdadmconfig"
	"github.com/linkease/quick-start/istore-backend/modules/raid/writecommands"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type raidMdadmConfigStore struct{}

func newRaidMdadmConfigService() *mdadmconfig.Service {
	return mdadmconfig.NewService(raidMdadmConfigStore{})
}

func (store raidMdadmConfigStore) LoadConfig(ctx context.Context) error {
	return uci.LoadConfig("mdadm", true)
}

func (store raidMdadmConfigStore) Arrays(ctx context.Context) []string {
	arrays, _ := uci.GetSections("mdadm", "array")
	return arrays
}

func (store raidMdadmConfigStore) DeleteFirstArray(ctx context.Context) error {
	return utils.BatchRun(ctx, writecommands.BuildDeleteFirstMdadmArrayCommand(), 0)
}

func (store raidMdadmConfigStore) Scan(ctx context.Context) (string, error) {
	stdout, _, err := utils.BatchOutErr(ctx, []string{"mdadm -Ds"}, 0)
	return stdout, err
}

func (store raidMdadmConfigStore) DiscoverMemberUUIDs(ctx context.Context) (string, error) {
	stdout, _, err := utils.BatchOutErr(ctx, []string{writecommands.BuildAutoFixUUIDCommand()}, 0)
	return stdout, err
}

func (store raidMdadmConfigStore) FindFreeMd(min int) int {
	return findFreeMd(min)
}

func (store raidMdadmConfigStore) AddArray(ctx context.Context, device string, uuid string) error {
	return utils.BatchRun(ctx, writecommands.BuildMdadmArrayCommands(device, uuid), 0)
}

func (store raidMdadmConfigStore) Commit(ctx context.Context) {
	utils.BatchRun(ctx, writecommands.BuildCommitMdadmCommand(), 0)
}

func (store raidMdadmConfigStore) Enable(ctx context.Context) {
	utils.BatchRun(ctx, writecommands.BuildEnableMdadmCommand(), 0)
}

func (store raidMdadmConfigStore) Restart(ctx context.Context) {
	utils.BatchRun(ctx, writecommands.BuildRestartMdadmCommand(), 0)
}
