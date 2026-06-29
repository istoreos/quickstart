package utils

import (
	"context"
	"fmt"
)

func UciCommitAndApply(ctx context.Context, configs []string) error {
	cmdList := []string{}
	for _, config := range configs {
		cmdList = append(cmdList,
			fmt.Sprintf("uci commit %v", config),
			fmt.Sprintf(`ubus call service event '{"type":"config.change", "data":{"package":"%v"}}'`, config),
		)
	}
	err := BatchRun(ctx, cmdList, 0)
	return err
}
