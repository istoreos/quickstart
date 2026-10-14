package service

import "time"

type LanHostRet struct {
	ip              string
	items           []*NetworkStatisticsItem
	txBytes         int64
	rxBytes         int64
	connectionCount int64
	sampledAt       time.Time
}

type lanStatsSnapshot struct {
	hosts     []*LanHostRet
	sampledAt time.Time
	err       error
	samples   int64
}
