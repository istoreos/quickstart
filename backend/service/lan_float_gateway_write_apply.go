package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/utils"
)

type LanFloatGatewayApply interface {
	Apply(ctx context.Context, configs []string) error
}

type defaultLanFloatGatewayApply struct{}

var lanFloatGatewayWriteCommitAndApply = func(ctx context.Context, configs []string) error {
	return utils.UciCommitAndApply(ctx, configs)
}

func NewDefaultLanFloatGatewayApply() LanFloatGatewayApply {
	return &defaultLanFloatGatewayApply{}
}

func (apply *defaultLanFloatGatewayApply) Apply(ctx context.Context, configs []string) error {
	_ = apply
	return lanFloatGatewayWriteCommitAndApply(ctx, configs)
}
