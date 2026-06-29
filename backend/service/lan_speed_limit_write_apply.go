package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/utils"
)

type LanSpeedLimitApply interface {
	Apply(ctx context.Context, configs []string) error
}

type defaultLanSpeedLimitApply struct{}

var lanSpeedLimitWriteCommitAndApply = func(ctx context.Context, configs []string) error {
	return utils.UciCommitAndApply(ctx, configs)
}

func NewDefaultLanSpeedLimitApply() LanSpeedLimitApply {
	return &defaultLanSpeedLimitApply{}
}

func (apply *defaultLanSpeedLimitApply) Apply(ctx context.Context, configs []string) error {
	_ = apply
	return lanSpeedLimitWriteCommitAndApply(ctx, configs)
}
