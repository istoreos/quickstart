package service

type FloatIPStatus struct {
	Installed       bool
	Enabled         bool
	Role            string
	SetIP           string
	CheckIP         string
	CheckURL        string
	CheckURLTimeout int64
}

type SpeedLimitStatus struct {
	Installed     bool
	Enabled       bool
	UploadSpeed   int64
	DownloadSpeed int64
}

type LanGlobalState struct {
	LanStatus       LanStatusSnapshot
	DhcpState       *LanDhcpState
	FloatIPState    FloatIPStatus
	SpeedLimitState SpeedLimitStatus
}
