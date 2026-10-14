//go:build !linux

// this is a dummy file for running tests on darwin, since conntrack is not supported on darwin, so we just return empty data

package service

import (
	"context"

	"github.com/istoreos/quickstart/backend/models"
)

type LanStats struct {
}

func NewLanStats() *LanStats {
	stats := &LanStats{}
	return stats
}

func (lstat *LanStats) reqHosts(_ string, _ bool) []*LanHostRet {
	return []*LanHostRet{}
}

func (lstat *LanStats) reqSnapshot(_ string, _ bool) lanStatsSnapshot {
	return lanStatsSnapshot{hosts: []*LanHostRet{}}
}

func (lstat *LanStats) reqSnapshotContext(ctx context.Context, _ string, _ bool) lanStatsSnapshot {
	if err := ctx.Err(); err != nil {
		return lanStatsSnapshot{err: err}
	}
	return lanStatsSnapshot{hosts: []*LanHostRet{}}
}

func (lstat *LanStats) diagnostics() *models.DeviceSamplerRuntimeDiagnostics {
	return &models.DeviceSamplerRuntimeDiagnostics{}
}
