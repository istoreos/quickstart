//go:build !linux

// this is a dummy file for running tests on darwin, since conntrack is not supported on darwin, so we just return empty data

package service

type LanStats struct {
}

func NewLanStats() *LanStats {
	stats := &LanStats{}
	return stats
}

func (lstat *LanStats) reqHosts(_ string, _ bool) []*LanHostRet {
	return []*LanHostRet{}
}
